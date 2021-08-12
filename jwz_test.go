package jwz

import (
	"fmt"
	"github.com/jhillyerd/enmime"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// Where we are going to store the emails. We know that the test data is of a fair size, so we tell the slice that
// in advance
//
var Emails = make([]Threadable, 0, 3000)

// TestMain sets up everything for the other test(s). It essentially parses a largish set of publicly available
// emails in to a structure that can then be used to perform email threading testing. To perform the parsing, we
// use the enmime package at https://github.com/jhillyerd/enmime
//
func TestMain(m *testing.M) {

	// Parse all the emails in the test directory
	//
	loadEmails()

	// OK, we have a fairly large email set all parsed, so now we can let the real tests run
	//
	os.Exit(m.Run())
}

func loadEmails() {
	_ = filepath.WalkDir("test/testdata", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			log.Printf("cannot process directory/file %s because: %#v", path, err)
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
					log.Printf("enmime parse yields severe error. File '%s', error: %#v", path, ee)
					return nil
				}
				log.Printf("enmime shows a non-fatal error. File '%s'm error: %#v", path, ee)
			}
		}

		// All is good, so let's accumulate the email
		//
		email := NewEmail(envelope)
		Emails = append(Emails, email)
		return nil
	})
}

// EmailRoot is a structure that implements the ThreadableRoot interface. I have not used ThreadableRoot
// here, but this is what it needs to look like if your input structure is not just a slice of Threadable
//
type EmailRoot struct {

	// This is some structure of the emails you want to thread, that you know how to traverse
	//
	emails []Threadable

	// You need some sort of position holder, which in this silly example is an index in the struct
	//
	position int
}

// Next sets the internal cursor to the next available Threadable
//
func (e *EmailRoot) Next() bool {
	e.position = e.position + 1
	if e.position < len(e.emails) {
		return true
	}
	return false
}

// Get returns the Threadable at the current internal cursor position
//
func (e *EmailRoot) Get() Threadable {
	return e.emails[e.position]
}

// NewThreadableRoot returns a struct instance that can be traversed using the ThreadableRoot interface
//
func NewThreadableRoot(emails []Threadable) ThreadableRoot {

	tr := &EmailRoot{
		emails:   emails,
		position: -1,
	}
	return tr
}

// Email is structure that implements the Threadable interface - this is what a user of this
// package needs to do.
//
type Email struct {
	email  *enmime.Envelope
	next   Threadable
	parent Threadable
	child  Threadable
	dummy  bool
	forID  string
}

func (e *Email) GetNext() Threadable {
	return e.next
}

func (e *Email) GetChild() Threadable {
	return e.child
}

// GetParent the parent Threadable of this node, if any
//
func (e *Email) GetParent() Threadable {
	return e.parent
}

func (e *Email) MessageThreadID() string {
	if e.dummy {
		return e.forID
	}
	return e.email.GetHeader("Message-Id")
}

func (e *Email) MessageThreadReferences() []string {
	refs := e.email.GetHeaderValues("References")
	return refs
}

var re = regexp.MustCompile("[Rr][Ee][ \t]*:[ \t]*")

func (e *Email) SimplifiedSubject() string {
	if e.dummy {
		return e.Subject()
	}
	subj := e.email.GetHeader("Subject")
	subj = re.ReplaceAllString(subj, "")
	return subj
}

func (e *Email) Subject() string {
	if e.dummy {
		if e.child != nil {
			return e.child.Subject() + " :: node synthesized by https://gatherstars.com/"
		}

		return fmt.Sprintf("Placeholder %s - manufactured by https://gatherstars.com/", e.forID)
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
	subj := e.email.GetHeader("Subject")
	return re.MatchString(subj)
}

func (e *Email) SetNext(next Threadable) {
	e.next = next
}

func (e *Email) SetChild(kid Threadable) {
	e.child = kid
	if kid != nil {
		kid.SetParent(e)
	}
}

// SetParent allows us to add or change the parent Threadable of this node
//
func (e *Email) SetParent(parent Threadable) {
	e.parent = parent
}

func (e *Email) MakeDummy(forID string) Threadable {
	return &Email{
		dummy: true,
		forID: forID,
	}
}

func (e *Email) IsDummy() bool {
	return e.dummy
}

func NewEmail(envelope *enmime.Envelope) Threadable {

	e := &Email{
		email: envelope,
		dummy: false,
	}
	return e
}

func ExampleThreader_ThreadSlice() {

	// emails := loadEmails() - your function to load emails into a slice
	//

	// Create a threader and thread using the slice of Threadable in the slice called Emails
	//
	threader := NewThreader()
	sliceRoot, err := threader.ThreadSlice(Emails)
	if err != nil {
		fmt.Printf("func ThreadSlice() error = %#v", err)
		return
	}

	// Make sure that number we got back, not including dummies, is the same as we sent in
	//
	var nc int
	Count(sliceRoot, &nc)
	if nc != len(Emails) {
		fmt.Printf("sent %d emails for threading, but got %d back", len(Emails), nc)
	} else {
		fmt.Printf("There are %d test emails", nc)
	}
	// Output: There are 2551 test emails
}

func TestThreader_ThreadSlice(t1 *testing.T) {

	// emails := loadEmails() - your function to load emails into a slice
	//

	// Create a threader and thread using the slice of Threadable in the slice called Emails
	//
	threader := NewThreader()
	sliceRoot, err := threader.ThreadSlice(Emails)
	if err != nil {
		t1.Errorf("func ThreadSlice() error = %#v", err)
	}

	// Make sure that number we got back, not including dummies, is the same as we sent in
	//
	var nc int
	Count(sliceRoot, &nc)
	if nc != len(Emails) {
		t1.Errorf("sent %d emails for threading, but got %d back", len(Emails), nc)
	}
}

func ExampleThreader_ThreadRoot() {

	// emails := loadEmails() - your function to load emails into a slice
	//

	// Create a threader and thread using the slice of Threadable in the slice called Emails
	//
	tr := NewThreadableRoot(Emails)
	threader := NewThreader()
	treeRoot, err := threader.ThreadRoot(tr)
	if err != nil {
		fmt.Printf("func ThreadRoot() error = %#v", err)
	}
	if treeRoot == nil {
		fmt.Printf("received no output from the threading algorithm")
	}
	// Make sure that number we got back, not including dummies, is the same as we sent in
	//
	var nc int
	Count(treeRoot, &nc)
	if nc != len(Emails) {
		fmt.Printf("sent %d emails for threading, but got %d back", len(Emails), nc)
	} else {
		fmt.Printf("There are %d test emails", nc)
	}
	// Output: There are 2551 test emails
}

func TestThreader_ThreadRoot(t1 *testing.T) {

	// emails := loadEmails() - your function to load emails into a slice
	//

	// Create a threader and thread using the ThreadableRootInterface to traverse the emails
	//
	tr := NewThreadableRoot(Emails)
	threader := NewThreader()
	treeRoot, err := threader.ThreadRoot(tr)
	if err != nil {
		t1.Errorf("ThreadRoot() error = %#v", err)
	}
	if treeRoot == nil {
		t1.Errorf("received no output from the threading algorithm")
	}
	// Make sure that number we got back, not including dummies, is the same as we sent in
	//
	var nc int
	Count(treeRoot, &nc)
	if nc != len(Emails) {
		t1.Errorf("Sent %d emails for threading, but got %d back", len(Emails), nc)
	}
}
