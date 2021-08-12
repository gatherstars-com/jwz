package main

import (
	"flag"
	"fmt"
	"github.com/gatherstars-com/jwz"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"log"
	"os"
	"sort"
	"strings"
)

func main() {

	var testData string
	var sizeHint int
	flag.StringVar(&testData, "emldir", "./eml", "base directory beneath which the test can find .eml files")
	flag.IntVar(&sizeHint, "sizehint", 3000, "supply a rough number of the emails you are scanning, if you have a clue")
	flag.Parse()

	// Use the enmime package to build all the emails we find under the given directory
	//
	threadables, err := buildEnvelopes(testData, sizeHint)
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
	x := jwz.Sort(sliceRoot, byDate)

	// And create a tree visualization in the console. Note that TERM must be set and your terminfo
	// stuff should be good. I don't know if this works on Windows - perhaps someone can tell me.
	//
	visualize(x)
}

// visualize a neat-ish ASCII tree representation of the threads, so we can do a human based check of whether
// things are working.
//
// Also, The colors will sometimes look wonky on a custom configuration of OSX console. Try a standard one and
// make sure your TERM and terminfo database are good if you have issues.
//
func visualize(threads jwz.Threadable) {

	root := tview.NewTreeNode("Threads").
		SetColor(tcell.ColorGreen)

	buildVisual(root, threads)

	tree := tview.NewTreeView().
		SetRoot(root).
		SetCurrentNode(root)

	sideBar := tview.NewTextView().
		SetTextAlign(tview.AlignLeft).
		SetText("Email Text")

	headers := tview.NewTextView().
		SetTextAlign(tview.AlignLeft).
		SetText("Email Headers").
		SetToggleHighlights(false)

	// There is only selected (mouse click or <enter>, not a
	tree.SetSelectedFunc(func(node *tview.TreeNode) {
		reference := node.GetReference()
		if reference == nil {
			sideBar.SetText("No email in this node")
		} else {
			thr := reference.(*Email)
			if thr.dummy {
				sideBar.SetText("Placeholders manufactured by the Threader don't contain email text")
			} else {
				sideBar.SetText(thr.email.Text)
				sideBar.ScrollToBeginning()
				headers.Clear()

				// Print each header in sorted order
				//
				hk := thr.email.GetHeaderKeys()
				sort.Strings(hk)
				for _, k := range hk {

					v := thr.email.GetHeader(k)
					if !strings.HasPrefix(k, "From ") {
						_, _ = headers.Write([]byte(fmt.Sprintf("%-30s: %s\n", k, v)))
						headers.ScrollToBeginning()
					}
				}
			}
		}
	})

	grid := tview.NewGrid().
		SetRows(-3, 0).
		SetColumns(-0, -2).
		SetBorders(true).
		AddItem(headers, 1, 0, 1, 2, 0, 0, false)

	grid.AddItem(tree, 0, 0, 1, 1, 5, 50, true).
		AddItem(sideBar, 0, 1, 1, 1, 0, 50, false)

	flex := tview.NewFlex().SetDirection(tview.FlexRow)

	flex.AddItem(tree, 0, 2, true)

	text := tview.NewBox().SetBorder(false)
	flex.AddItem(text, 0, 3, false)

	info := tview.NewBox().SetBorder(true)
	flex.AddItem(info, 7, 1, false)

	app := tview.NewApplication().SetRoot(grid, true).EnableMouse(true)

	if err := app.Run(); err != nil {
		panic(err)
	}
}

// This function is called when an email node is selected in the tree
//

// buildVisual creates a visual node for the given treeNode and makes it a child of the
// given target node.
//
func buildVisual(target *tview.TreeNode, treeNode jwz.Threadable) {

	if treeNode == nil {
		// We have descended far enough in this part of the tree
		//
		return
	}

	// We need a new visual node for this part of the tree
	//
	n := tview.NewTreeNode(tview.Escape(treeNode.Subject())).
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
	buildVisual(n, treeNode.GetChild())

	// But all the next links in the chain from this node are children of our current visual
	//
	for next := treeNode.GetNext(); next != nil; next = next.GetNext() {

		// So we create a visual node for this next email
		//
		nn := tview.NewTreeNode(next.Subject()).
			SetReference(next)

		// Select a color
		//
		colorMe(nn, next)

		// And it becomes a child of the target too
		//
		target.AddChild(nn)

		// And in turn, we need to add its children
		//
		buildVisual(nn, next.GetChild())
	}
}

// colorMe decides what color a visualized node should be, based on whether it has child and/or next
//
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
		n.SetColor(tcell.ColorLightYellow)
	}
}
