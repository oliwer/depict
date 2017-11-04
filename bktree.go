// Adapted from https://raw.githubusercontent.com/DrakeW/go-bk-tree

// Package go_bk_tree is a tree data structure (implemented in Golang) specialized to index data in a metric space.
// The BK-tree data structure was proposed by Burkhard and Keller in 1973 as a solution to the problem of
// searching a set of keys to find a key which is closest to a given query key. (Doc reference: http://signal-to-noise.xyz/post/bk-tree/)
package main

import "sync"

type bkTreeNode struct {
	Image ImageInfo
	Children map[int]*bkTreeNode
}

func newbkTreeNode(ii ImageInfo) *bkTreeNode {
	return &bkTreeNode{
		Image: ii,
		Children: make(map[int]*bkTreeNode),
	}
}

type BKTree struct {
	sync.Mutex
	Root *bkTreeNode
}

// Add a node to BK-Tree, the location of the new node
// depends on how distance between different tensors are defined
func (tree *BKTree) Add(val ImageInfo) {
	node := newbkTreeNode(val)
	tree.Lock()
	defer tree.Unlock()
	if tree.Root == nil {
		tree.Root = node
		return
	}
	curNode := tree.Root
	for {
		dist := curNode.Image.DistanceFrom(val)
		target := curNode.Children[dist]
		if target == nil {
			curNode.Children[dist] = node
			break
		}
		curNode = target
	}
}

func (tree *BKTree) Search(val ImageInfo, radius int) []ImageInfo {
	if tree.Root == nil {
		return []ImageInfo{}
	}

	candidates := make([]*bkTreeNode, 0, 10)
	candidates = append(candidates, tree.Root)
	results := make([]ImageInfo, 0, 5)

	for {
		cand := candidates[0]
		candidates = candidates[1:]

		dist := cand.Image.DistanceFrom(val)

		if dist <= radius && cand.Image.Name != val.Name {
			results = append(results, cand.Image)
		}

		low, high := dist-radius, dist+radius
		for dist, child := range cand.Children {
			if dist >= low && dist <= high {
				candidates = append(candidates, child)
			}
		}
		if len(candidates) == 0 {
			break
		}
	}
	return results
}

func (tree *BKTree) SearchByName(name string) *ImageInfo {
	if tree.Root == nil {
		return nil
	}

	candidates := make([]*bkTreeNode, 0, 10)
	candidates = append(candidates, tree.Root)

	for {
		cand := candidates[0]
		candidates = candidates[1:]

		if cand.Image.Name == name {
			return &cand.Image
		}

		for _, child := range cand.Children {
			candidates = append(candidates, child)
		}
		if len(candidates) == 0 {
			break
		}
	}

	return nil
}

func (tree *BKTree) SearchSimilars(radius int) map[string][]string {
	if tree.Root == nil {
		return nil
	}

	candidates := make([]*bkTreeNode, 0, 2048)
	candidates = append(candidates, tree.Root)
	results := make(map[string][]string, 64)

	for {
		cand := candidates[0]
		candidates = candidates[1:]

		for _, ii := range tree.Search(cand.Image, radius) {
			// Avoid duplicates
			_, found := results[ii.Name]
			if !found {
				results[cand.Image.Name] =
					append(results[cand.Image.Name], ii.Name)
			}
		}

		for _, child := range cand.Children {
			candidates = append(candidates, child)
		}
		if len(candidates) == 0 {
			break
		}
	}

	return results
}
