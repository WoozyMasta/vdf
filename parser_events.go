// SPDX-License-Identifier: MIT
// Copyright (c) 2026 Maxim Levchenko (WoozyMasta)
// Source: github.com/woozymasta/vdf

package vdf

// eventFrame stores traversal progress for one node.
type eventFrame struct {
	node       *Node // Node being processed.
	childIndex int   // Index of the current child being processed.
	started    bool  // Whether the node has been started.
}

// eventIterator iterates through document events lazily without prebuilding slices.
type eventIterator struct {
	doc       *Document    // Document being processed.
	stack     []eventFrame // Stack of event frames.
	rootIndex int          // Index of the current root being processed.
	started   bool         // Whether the document has been started.
	finished  bool         // Whether the document has been finished.
}

// newEventIterator creates a DFS event iterator for a document.
func newEventIterator(doc *Document) *eventIterator {
	it := &eventIterator{
		doc: doc,
	}

	if doc != nil {
		it.stack = make([]eventFrame, 0, len(doc.Roots)+4)
	}

	return it
}

// next returns next event and false once stream is exhausted.
func (it *eventIterator) next() (Event, bool) {
	if it.finished {
		return Event{}, false
	}

	if !it.started {
		it.started = true
		return Event{Type: EventDocumentStart, Depth: 0}, true
	}

	// Process the document until it is finished.
	for {
		if len(it.stack) == 0 {
			if it.doc == nil || it.rootIndex >= len(it.doc.Roots) {
				it.finished = true
				return Event{Type: EventDocumentEnd, Depth: 0}, true
			}

			root := it.doc.Roots[it.rootIndex]
			it.rootIndex++
			if root == nil {
				continue
			}

			it.stack = append(it.stack, eventFrame{node: root})
			continue
		}

		topIndex := len(it.stack) - 1
		frame := &it.stack[topIndex]
		depth := len(it.stack)

		// Get the node kind and process it.
		switch frame.node.Kind {
		case NodeObject:
			if !frame.started {
				frame.started = true
				return Event{Type: EventObjectStart, Key: frame.node.Key, Depth: depth}, true
			}

			for frame.childIndex < len(frame.node.Children) {
				child := frame.node.Children[frame.childIndex]
				frame.childIndex++
				if child == nil {
					continue
				}

				it.stack = append(it.stack, eventFrame{node: child})
				break
			}

			if len(it.stack) > topIndex+1 {
				continue
			}

			it.stack = it.stack[:topIndex]
			return Event{Type: EventObjectEnd, Key: frame.node.Key, Depth: depth}, true

		case NodeString:
			it.stack = it.stack[:topIndex]
			return Event{Type: EventString, Key: frame.node.Key, Depth: depth, StringValue: frame.node.StringValue}, true

		case NodeUint32:
			it.stack = it.stack[:topIndex]
			return Event{Type: EventUint32, Key: frame.node.Key, Depth: depth, Uint32Value: frame.node.Uint32Value}, true

		default:
			// Unknown node kind is skipped here; document-level validation guards this path.
			it.stack = it.stack[:topIndex]
		}
	}
}
