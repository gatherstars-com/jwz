package main

import (
	"fmt"
	"github.com/gatherstars-com/jwz"
	"github.com/jhillyerd/enmime"
	"io/fs"
	"log"
	"net/mail"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// buildEnvelopes finds all email files beneath the given emlDir (files starting with .eml), parses them
// and returns a slice full of all the envelopes that we parsed in to existence
//
func buildEnvelopes(emlDir string, sizeHint int) ([]jwz.Threadable, error) {

	duprem := make(map[string]string)
	// Where we are going to store the emails. We know that the test data is of a fair size, so we tell the slice that
	// in advance
	//
	var emails = make([]jwz.Threadable, 0, sizeHint)

	_ = filepath.WalkDir(emlDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			log.Printf("cannot process file %s because: %s", path, err)
			os.Exit(1)
		}

		if d.IsDir() {

			// Skip any directory entries, including the base test data dir
			//
			return nil
		}

		// We only look at files that have an eml extension
		//
		if !strings.HasSuffix(path, ".eml") {
			return nil
		}

		f, e := os.Open(path)
		if e != nil {
			return err
		}
		envelope, e1 := enmime.ReadEnvelope(f)
		_ = f.Close()
		if e1 != nil {
			log.Printf("cannot parse email file error = %v", e1)
			return nil
		}

		// See if the parser thinks that there are errors - the error recording in the package
		// is unfortunately a little hokey in that we just get some strings and will have to do
		// lots of processing if it is left that way. I would prefer to contribute some code that
		// processes errors like the AWS SDK, so we can check the types of error/warning etc.
		//
		if len(envelope.Errors) > 0 {
			for _, ee := range envelope.Errors {
				if !ee.Severe {
					log.Printf("enmime shows a non-fatal error. File '%s', error: %#v", path, ee)
				} else {
					log.Printf("enmime parse yields severe error. File: '%s', error: %#v", path, ee)
					return nil
				}
			}
		}

		// All is good, so let's accumulate the email, unless it's a duplicate, which sometimes happens
		// with spammers and test email sets from Kaggle. We will keep duplicates in the input directory but ignore
		// them for processing in this example. The actual unit test processes all the emails in testdata so that
		// dealing with garbage input is tested, but here we are creating a visual, so let's just remove the
		// Manchester United.
		//
		thisID := envelope.GetHeader("Message-Id")
		ent, present := duprem[thisID]
		if present {
			log.Printf("Duplicate message id in file '%s' ignored in favor of file '%s'", path, ent)
		} else {
			duprem[thisID] = path
			email := NewEmail(envelope)
			emails = append(emails, email)
		}
		return nil
	})

	return emails, nil
}

// Email implements the Threadable interface - see the interface documentation for the function docs.
// This is what a user of this package needs to provide to thread their messages, though you don't have to use
// enmime of course.
//
// You can put anything in here as only next and child are required, though you need some sort of
// reference to the email, or at least a few parts of it, such as the Subject, Message-Id, and References
// headers.
//
// You could also add a prev field if you need doubly linked next lists, which can make some things easier
// etc. Or use a container or something. This is the simplest example - you should get the idea, big nose.
//
type Email struct {
	email  *enmime.Envelope
	next   jwz.Threadable
	parent jwz.Threadable
	child  jwz.Threadable
	dummy  bool
	forID  string
}

// GetNext the next Threadable in the chain, if any
//
func (e *Email) GetNext() jwz.Threadable {
	return e.next
}

// GetChild the child Threadable of this node, if any
//
func (e *Email) GetChild() jwz.Threadable {
	return e.child
}

// GetParent the parent Threadable of this node, if any
//
func (e *Email) GetParent() jwz.Threadable {
	return e.parent
}

// GetDate extracts the timestamp from the enmime envelope contained in the supplied Threadable
//
func (e *Email) GetDate() time.Time {

	// We can have dummies because we are likely to have parsed a set of emails with incomplete threads,
	// where the start of the thread or sub thread was referenced, but we did not get to parse it, at least yet.
	// This means it will be a placeholder as the root for the thread, so we can use the time of the child as the
	// time of this email.
	//
	if e.IsDummy() {
		if e.GetChild() != nil {
			return e.GetChild().GetDate()
		}

		// Protect against having nothing in the children that knows what time it is. So, back to the
		// beginning of time according to Unix
		//
		return time.Unix(0, 0)
	}
	emailDateStr := e.email.GetHeader("Date")
	d, err := mail.ParseDate(emailDateStr)
	if err != nil {
		return time.Unix(0, 0)
	}
	return d
}

// MessageThreadID the id of this email message
//
func (e *Email) MessageThreadID() string {
	if e.dummy {
		return e.forID
	}
	return e.email.GetHeader("Message-Id")
}

// MessageThreadReferences the list of references mentioned in dispatches
//
func (e *Email) MessageThreadReferences() []string {
	if e.dummy {
		return nil
	}

	refs := e.email.GetHeaderValues("References")
	return refs
}

var re = regexp.MustCompile("[Rr][Ee][ \t]*:[ \t]*")

// SimplifiedSubject the email subject without any Re: prefixes. If this
// Threadable is a dummy, but it has a child, we try to use the subject in the
// child, if it has one. Note that this is only used for threading if there are
// no references etc
//
func (e *Email) SimplifiedSubject() string {
	if e.dummy {
		return e.Subject()
	}
	subj := e.email.GetHeader("Subject")
	subj = re.ReplaceAllString(subj, "")
	return strings.Trim(subj, " ")
}

// Subject the subject as defined in the email, plus a few adornments so that humans can see that
// the threading is correct and see the sort order
//
func (e *Email) Subject() string {

	if e.dummy {
		if e.child != nil {
			return e.child.Subject() + " :: node synthesized by https://gatherstars.com/"
		}

		return fmt.Sprintf("Placeholder %s - manufactured by https://gatherstars.com/", e.MessageThreadID())
	}

	// Add in the date for a bit of extra information
	//
	var sb strings.Builder
	t := e.GetDate()
	sb.WriteString(t.UTC().String())
	sb.WriteString(" : ")
	sb.WriteString(strings.Trim(e.email.GetHeader("Subject"), " "))
	return sb.String()
}

// SubjectIsReply true if the Subject header of this email appears to indicate it is a reply to something.
// This is only used if there are no references in the Threadables that otherwise show that relationship
func (e *Email) SubjectIsReply() bool {
	if e.dummy {
		return false
	}
	subj := e.email.GetHeader("Subject")
	return re.MatchString(subj)
}

// SetNext allows us to change or add a pointer to the next Threadable in a set of emails
//
func (e *Email) SetNext(next jwz.Threadable) {
	e.next = next
}

// SetChild allows us to add or change the child Threadable of this node
//
func (e *Email) SetChild(kid jwz.Threadable) {
	e.child = kid
	if kid != nil {
		kid.SetParent(e)
	}
}

// SetParent allows us to add or change the parent Threadable of this node
//
func (e *Email) SetParent(parent jwz.Threadable) {
	e.parent = parent
}

// MakeDummy manufactures a placeholder Threadable that other Threadables can become children of. See
// interface documentation for more details - note that this may be subject to change to also supply the
// MessageID that this dummy is placeholder for, if we have it
//
func (e *Email) MakeDummy(forID string) jwz.Threadable {
	return &Email{
		dummy: true,
		forID: forID,
	}
}

// IsDummy true if this Threadable is a Manchester United supporter
//
func (e *Email) IsDummy() bool {
	return e.dummy
}

// NewEmail creates a new structure that implements the jwz.Threadable interface. it should be obvious that
// your struct can contain whatever you want it to, so long as it obeys the jwz.Threadable contract. Here,
// we use it to pass around a pointer to the enmime envelope that we have parsed an email into.
//
func NewEmail(envelope *enmime.Envelope) jwz.Threadable {

	e := &Email{
		email: envelope,
		dummy: false,
	}
	return e
}

// byDate is a sort comparison function that is used to call in to the Threadable Sort utility function and
// sort the threads by Date
//
func byDate(t1 jwz.Threadable, t2 jwz.Threadable) bool {

	return t1.GetDate().Before(t2.GetDate())
}
