package main

import (
	"flag"
	"github.com/gatherstars-com/jwz/pkg/jwz"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"log"
	"os"
	"sort"
)

func main() {

	var testData string
	var sizeHint int
	flag.StringVar(&testData, "emldir", "./eml", "base directory beneath which the test can find .eml files")
	flag.IntVar(&sizeHint, "sizehint", 3000, "supply a rough number of the emails you are scanning, if you have a clue")
	flag.Parse()

	// Use the enmime package to build all the emails we find under the given directory
	//
	threadables, err := BuildEnvelopes(testData, sizeHint)
	if err != nil {
		log.Printf("Unable to walk the eml directory: %#v", err)
		os.Exit(1)
	}

	// Now we have a big slice of all the emails, lets use our jwz algorithm to place them in to a thread tree
	//
	threader := jwz.NewThreader()
	sliceRoot, err := threader.ThreadSlice(threadables)
	if err != nil {
		log.Fatalf("Email threading operation return fatal error: %#v", err)
	}

	// Sort it Rodney!
	//
	x := sortByDate(sliceRoot)

	// And create a tree visualization in the console. Note that TERM must be set and your terminfo
	// stuff should be good. I don't know if this works on Windows - perhaps someone can tell me.
	//
	visualize(x)
}

// Send a neatish ASCII tree representation of the threads, so we can do a human based check of whether
// things are working. Note that there is no sort of the tree implemented - you have to do that yourself -
// so even though you might run it against the same data twice, the display will be in a different order because
// go maps do not repeat order, deliberately.
//
// Also, I have not been able to get colors to work correctly in OSX console with the tview for some reason,
// but this is an example of jwz, not tview
//
func visualize(threads jwz.Threadable) {

	root := tview.NewTreeNode("Threads").
		SetColor(tcell.ColorGreen)

	addNode(root, threads)

	tree := tview.NewTreeView().
		SetRoot(root).
		SetCurrentNode(root)
	if err := tview.NewApplication().SetRoot(tree, true).EnableMouse(true).Run(); err != nil {
		panic(err)
	}
}

// sortByDate is a fixed sort by date only just to give the general idea to people. Someone might want to
// submit a sorting interface to the repo as a contribution. Maybe I will do it in the future.
//
func sortByDate(treeNode jwz.Threadable) jwz.Threadable {

	// Guard against stupidity
	//
	if treeNode == nil {
		return nil
	}

	// If this node has no next pointers, then it is the only element in this current chain, so it is
	// sorted by default
	//
	if treeNode.GetNext() == nil {
		return treeNode
	}

	// Now we sort the chain of current pointers. The easiest way is to convert to a slice then
	// sort the slice. This is because after sorting, the child of the node above should become the
	// first element of the sorted slice
	//
	var s = make([]jwz.Threadable, 0, 10)
	for current := treeNode; current != nil; current = current.GetNext() {

		// If the current node we are inspecting has a child, then sort that first
		//
		if c := current.GetChild(); c != nil {
			treeNode.SetChild(sortByDate(c))
		}
		s = append(s, current)
	}

	// We can now sort the slice of next pointers at this level
	//
	sort.Slice(s, func(i, j int) bool {
		return GetEmailDate(s[i]).Before(GetEmailDate(s[j]))
	})

	// And we now rebuild the chain from the slice
	//
	l := len(s) - 1
	for i := 0; i < l; i++ {
		s[i].SetNext(s[i+1])
	}

	// Last element in the slice no longer has a current of course
	//
	s[l].SetNext(nil)

	// And the new child of the node above is the first of the newly sorted (or at the top level the new root)
	//
	newChild := s[0]
	s = nil

	// And return the new child for the node above us
	//
	return newChild
}

func addNode(target *tview.TreeNode, treeNode jwz.Threadable) {

	if treeNode == nil {
		// We have descended far enough in this part of the tree
		//
		return
	}

	// We need a new visual node for this part of the tree
	//
	n := tview.NewTreeNode(treeNode.Subject()).
		SetReference(treeNode)

	// And select a colour
	//
	colorMe(n, treeNode)

	// Which we add to the target node
	//
	target.AddChild(n)

	// Then we add the child of this node to newly create visual entry as a child. This recursive call will make sure we
	// descend the children of every node correctly as we trace the next pointers at the same level after this
	//
	addNode(n, treeNode.GetChild())

	// But all the next links in the chain from this node are children of our current visual
	//
	for next := treeNode.GetNext(); next != nil; next = next.GetNext() {

		// So we create a visual node for this next email
		//
		nn := tview.NewTreeNode(next.Subject())

		// Select a color
		//
		colorMe(nn, next)

		// And it becomes a child of the target too
		//
		target.AddChild(nn)

		// And in turn, we need to add its children
		//
		addNode(nn, next.GetChild())
	}
}

func colorMe(n *tview.TreeNode, treeNode jwz.Threadable) {

	hasNext := treeNode.GetNext() != nil
	hasChild := treeNode.GetChild() != nil

	// Color according to taste - not that the background color of your terminal and whether you have bothered
	// to set TERM and terminfo up correctly may make these color selections look strange or even not work
	//
	switch {

	case hasNext && hasChild:

		// With a next and a child, it is part of a thread, but also the start of a sub thread
		//
		n.SetColor(tcell.ColorLightGreen)

	case hasNext:

		// With a next and no child, it is a reply by some boring old gammon
		//
		n.SetColor(tcell.ColorLightBlue)

	case hasChild:

		// With a child and no next then it is the start of a sub-thread
		//
		n.SetColor(tcell.ColorLightGreen)

	default:

		// With no child and no next, then it is the haggard old spinster of the email world
		// and either resides on its own, or has the last word on a thread. Or something.
		//
		n.SetColor(tcell.ColorLightBlue)
	}
}
