package jwz

import (
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
var emails = make([]Threadable, 0, 3000)

// TestMain sets up everything for the other test(s). It essentially parses a largish set of publicly available
// emails in to a structure that can then be used to perform email threading testing. To perform the parsing, we
// use the enmime package at https://github.com/jhillyerd/enmime
//
func TestMain(m *testing.M) {

	_ = filepath.WalkDir("../../test/testdata", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			log.Printf("cannot process file %s because: %#v", path, err)
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
				} else {
					log.Printf("enmime shows a non-fatal error. File '%s'm error: %#v", path, ee)
				}
			}
		}

		// All is good, so let's accumulate the email
		//
		email := NewEmail(envelope)
		emails = append(emails, email)
		return nil
	})

	// OK, we have a fairly large email set all parsed, so now we can let the real tests run
	//
	os.Exit(m.Run())
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
	} else {
		return false
	}
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
	child  Threadable
	dummy  bool
	dCount int
}

func (e *Email) GetNext() Threadable {
	return e.next
}

func (e *Email) GetChild() Threadable {
	return e.child
}

func (e *Email) MessageThreadID() string {
	return e.email.GetHeader("Message-Id")
}

func (e *Email) MessageThreadReferences() []string {
	refs := e.email.GetHeaderValues("References")
	return refs
}

var re = regexp.MustCompile("[Rr][Ee][ \t]*:[ \t]*")

func (e *Email) SimplifiedSubject() string {

	subj := e.email.GetHeader("Subject")
	subj = re.ReplaceAllString(subj, "")
	return subj
}

func (e *Email) Subject() string {
	return e.email.GetHeader("Subject")
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
}

func (e *Email) MakeDummy(count int) Threadable {
	return &Email{
		dummy:  true,
		dCount: count,
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

func TestThreader_ThreadSlice(t1 *testing.T) {

	threader := NewThreader()
	sliceRoot, err := threader.ThreadSlice(emails)
	if err != nil {
		t1.Errorf("func ThreadSlice() error = %#v", err)
	}

	// Make sure that number we got back, not including dummies, is the same as we sent in
	//
	var nc int
	count(sliceRoot, &nc)
	if nc != len(emails) {
		t1.Errorf("Sent %d emails for threading, but got %d back", len(emails), nc)
	}
}

func TestThreader_ThreadRoot(t1 *testing.T) {

	tr := NewThreadableRoot(emails)
	threader := NewThreader()
	treeRoot, err := threader.ThreadRoot(tr)
	if err != nil {
		t1.Errorf("ThreadRoot() error = %#v", err)
	}
	if treeRoot == nil {
		t1.Errorf("Received no output from the threading algorithm")
	}
	// Make sure that number we got back, not including dummies, is the same as we sent in
	//
	var nc int
	count(treeRoot, &nc)
	if nc != len(emails) {
		t1.Errorf("Sent %d emails for threading, but got %d back", len(emails), nc)
	}
}

func count(root Threadable, counter *int) {

	if root == nil {
		return
	}

	for node := root; node != nil; node = node.GetNext() {

		if c := node.GetChild(); c != nil {

			// Count children of the current one first then
			//
			count(c, counter)
		}

		// Only count this one if it is not a dummy placeholder
		//
		if !node.IsDummy() {
			*counter++
		}
	}
}
