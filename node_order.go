// SPDX-License-Identifier: MIT
// Copyright (c) 2026 Maxim Levchenko (WoozyMasta)
// Source: github.com/woozymasta/vdf

package vdf

import "slices"

// orderedNodes returns nodes in source order or deterministic key order.
func orderedNodes(in []*Node, deterministic bool) []*Node {
	if !deterministic {
		return in
	}

	out := make([]*Node, len(in))
	copy(out, in)

	// Stable sort keeps relative order for equal keys, preserving duplicate-key sequences.
	slices.SortStableFunc(out, func(a, b *Node) int {
		if a == nil && b == nil {
			return 0
		}
		if a == nil {
			return 1
		}
		if b == nil {
			return -1
		}
		if a.Key < b.Key {
			return -1
		}
		if a.Key > b.Key {
			return 1
		}

		return 0
	})

	return out
}
