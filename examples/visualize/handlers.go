package main

import (
	"fmt"
	"github.com/gatherstars-com/jwz/pkg/jwz"
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

// BuildEnvelopes finds all email files beneath the given emlDir (files starting with .eml), parses them
// and returns a slice full of all the envelopes that we parsed in to existence
//
func BuildEnvelopes(emlDir string, sizeHint int) ([]jwz.Threadable, error) {

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
				if ee.Severe {
					log.Printf("enmime parse yields severe error. File: '%s', error: %#v", path, ee)
					return nil
				} else {
					log.Printf("enmime shows a non-fatal error. File '%s', error: %#v", path, ee)
				}
			}
		}

		// All is good, so let's accumulate the email, unless it's a duplicate, which sometimes happens
		// with spammers and test email sets from Kagel. We will keep duplicates in the input directory but ignore
		// them for processing in this example. The actual unit test processes all the emails in testdata so that
		// dealing with garbage input is tested, but here we are creating a visual, so let's just remove the
		// Manchester United.
		//
		thisId := envelope.GetHeader("Message-Id")
		ent, present := duprem[thisId]
		if present {
			log.Printf("Duplicate message id in file '%s' ignored in favor of file '%s'", path, ent)
		} else {
			duprem[thisId] = path
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
// You could also add a parent and prev field if you need doubly linked lists, which can make sorting
// easier etc. Or use a container or something. This is the simplest example - you should get the idea, big nose.
//
type Email struct {
	email  *enmime.Envelope
	next   jwz.Threadable
	child  jwz.Threadable
	dummy  bool
	dcount int
}

func (e *Email) GetNext() jwz.Threadable {
	return e.next
}

func (e *Email) GetChild() jwz.Threadable {
	return e.child
}

func (e *Email) MessageThreadID() string {
	if e.dummy {
		return fmt.Sprintf("<fake-id-%d>", e.dcount)
	}
	return e.email.GetHeader("Message-Id")
}

func (e *Email) MessageThreadReferences() []string {
	if e.dummy {
		return nil
	}

	refs := e.email.GetHeaderValues("References")
	return refs
}

var re = regexp.MustCompile("[Rr][Ee][ \t]*:[ \t]*")

func (e *Email) SimplifiedSubject() string {
	if e.dummy {
		return fmt.Sprintf("Subject %d", e.dcount)
	}
	subj := e.email.GetHeader("Subject")
	subj = re.ReplaceAllString(subj, "")
	return strings.Trim(subj, " ")
}

func (e *Email) Subject() string {

	if e.dummy {
		if e.child != nil {
			return e.child.Subject() + " :: node synthesized by https://gatherstars.com/"
		} else {
			return fmt.Sprintf("Placeholder %d - manufactured by https://gatherstars.com/", e.dcount)
		}
	}

	// Add in the date for a bit of extra information
	//
	var sb strings.Builder
	t := GetEmailDate(e)
	sb.WriteString(t.UTC().String())
	sb.WriteString(" : ")
	sb.WriteString(strings.Trim(e.email.GetHeader("Subject"), " "))
	return sb.String()
}

func (e *Email) SubjectIsReply() bool {
	if e.dummy {
		return false
	}
	subj := e.email.GetHeader("Subject")
	return re.MatchString(subj)
}

func (e *Email) SetNext(next jwz.Threadable) {
	e.next = next
}

func (e *Email) SetChild(kid jwz.Threadable) {
	e.child = kid
}

func (e *Email) MakeDummy(count int) jwz.Threadable {
	return &Email{
		dummy:  true,
		dcount: count,
	}
}

func (e *Email) IsDummy() bool {
	return e.dummy
}

func NewEmail(envelope *enmime.Envelope) jwz.Threadable {

	e := &Email{
		email: envelope,
		dummy: false,
	}
	return e
}

// GetEmailDate extracts the timestamp from the enmime envelope contained in the supplied Threadable
//
func GetEmailDate(e jwz.Threadable) time.Time {

	// We can have dummies because we are likely to have parsed a set
	// of emails with incomplete threads, where the start of the thread or sub thread was referenced, but
	// we did not get to parse it. This means it will be a placeholder as the root for the thread, so
	// we can use the time of the child as the time of this email. Note that when sorting, the child of
	// this node will already have been sorted before the chain that this is a part of, so the child will
	// be the correct one in sort sequence
	//
	if e.IsDummy() {
		if e.GetChild() != nil {
			return GetEmailDate(e.GetChild())
		} else {

			// Protect against bugs
			//
			return time.Unix(0, 0)
		}
	}
	envelope := e.(*Email).email
	emailDateStr := envelope.GetHeader("Date")
	d, err := mail.ParseDate(emailDateStr)
	if err != nil {
		return time.Unix(0, 0)
	}
	return d
}
