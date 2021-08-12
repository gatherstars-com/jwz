package jwz

import (
	"fmt"
	"net/mail"
	"time"
)

func ExampleSort() {

	// emails := loadEmails() - your function to load emails into a slice
	//

	// Create a threader and thread using the slice of Threadable in the slice called Emails
	//
	threader := NewThreader()
	sliceRoot, err := threader.ThreadSlice(Emails)
	if err != nil {
		fmt.Printf("func ThreadSlice() error = %#v", err)
	}

	// Count with no dummies
	//
	var n, ns int
	Count(sliceRoot, &n)

	// Sort it Rodney!
	//
	x := Sort(sliceRoot, func(t1 Threadable, t2 Threadable) bool {

		// Sort by date, inline function
		//
		return GetEmailDate(t1).Before(GetEmailDate(t2))
	})

	Count(x, &ns)
	if n != ns {
		fmt.Printf("Count before sort: %d, Count after sort: %d\n", n, ns)
	}

	fmt.Printf("First node subject: %s\n", x.Subject())

	// Output: First node subject: 1970-01-01 00:00:00 +0000 UTC : Re: My source: RE: A biblical digression :: node synthesized by https://gatherstars.com/
}

// GetEmailDate extracts the timestamp from the enmime envelope contained in the supplied Threadable
//
func GetEmailDate(e Threadable) time.Time {

	// We can have dummies because we are likely to have parsed a set of emails with incomplete threads,
	// where the start of the thread or sub thread was referenced, but we did not get to parse it, at least yet.
	// This means it will be a placeholder as the root for the thread, so we can use the time of the child as the
	// time of this email.
	//
	if e.IsDummy() {
		if e.GetChild() != nil {
			return GetEmailDate(e.GetChild())
		}

		// Protect against having nothing in the children that knows what time it is. So, back to the
		// beginning of time according to Unix
		//
		return time.Unix(0, 0)
	}
	envelope := e.(*Email).email
	emailDateStr := envelope.GetHeader("Date")
	d, err := mail.ParseDate(emailDateStr)
	if err != nil {
		return time.Unix(0, 0)
	}
	return d
}

func ExampleWalk_depth() {

	// emails := loadEmails() - your function to load emails into a slice
	//

	// Create a threader and thread using the slice of Threadable in the slice called Emails
	//
	threader := NewThreader()
	sliceRoot, err := threader.ThreadSlice(Emails)
	if err != nil {
		fmt.Printf("func ThreadSlice() error = %#v", err)
	}
	var c int
	_ = Walk(true, sliceRoot, func(t Threadable, u interface{}) (bool, error) {

		c := u.(*int)
		if !t.IsDummy() {
			*c++
		}
		return false, nil
	},
		&c)

	fmt.Printf("Walker walked %d depth first\n", c)

	// Output: Walker walked 2551 depth first
}

func ExampleWalk_breadth() {

	// emails := loadEmails() - your function to load emails into a slice
	//

	// Create a threader and thread using the slice of Threadable in the slice called Emails
	//
	threader := NewThreader()
	sliceRoot, err := threader.ThreadSlice(Emails)
	if err != nil {
		fmt.Printf("func ThreadSlice() error = %#v", err)
	}
	var c int

	// Walk the tree breadth first and call our anonymous function on each Threadable
	//
	_ = Walk(false, sliceRoot, func(t Threadable, u interface{}) (bool, error) {

		c := u.(*int)
		if !t.IsDummy() {
			*c++
		}
		return false, nil
	},
		&c)

	fmt.Printf("Walker walked %d depth first\n", c)

	// Output: Walker walked 2551 depth first
}

type searcher struct {
	messageID string
	e         Threadable
}

func ExampleWalk_search() {

	// emails := loadEmails() - your function to load emails into a slice
	//

	// Create a threader and thread using the slice of Threadable in the slice called Emails
	//
	threader := NewThreader()
	sliceRoot, err := threader.ThreadSlice(Emails)
	if err != nil {
		fmt.Printf("func ThreadSlice() error = %#v", err)
	}

	// Construct our search
	//
	var param = &searcher{
		messageID: "<008701c24a16$e3443830$0200a8c0@JMHALL>",
	}

	// Walk the tree breadth first and call our anonymous function on each Threadable
	//
	_ = Walk(false, sliceRoot,
		func(t Threadable, u interface{}) (bool, error) {

			params := u.(*searcher)
			if !t.IsDummy() {
				if t.MessageThreadID() == params.messageID {

					// We found the email we wanted, so we can stop here
					//
					params.e = t
					return true, nil
				}
			}
			return false, nil
		},
		param)

	fmt.Printf("Walker found the email %s with subject %s\n", param.messageID, param.e.Subject())

	// Output: Walker found the email <008701c24a16$e3443830$0200a8c0@JMHALL> with subject Property rights in the 3rd World (De Soto's Mystery of Capital)
}

func ExampleCount() {

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

	// Find out how many non dummy Threadables are in the tree - in other words, how many
	// actual emails are there in the tree?
	//
	var nc int
	Count(sliceRoot, &nc)
	fmt.Printf("There are %d test emails", nc)
	// Output: There are 2551 test emails
}
