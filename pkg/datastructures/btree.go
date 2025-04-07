package datastructures

import (
	"slices"
	"sync"
)

// BTreeNode represents a node in the B-tree
type BTreeNode struct {
	keys     []string
	values   []any
	children []*BTreeNode
	leaf     bool
}

// BTree is a B-tree implementation for efficient indexing
type BTree struct {
	root     *BTreeNode
	degree   int
	size     int
	mu       sync.RWMutex
}

// NewBTree creates a new B-tree with the specified degree
func NewBTree(degree int) *BTree {
	if degree < 2 {
		degree = 2 // Minimum degree is 2
	}

	return &BTree{
		root:   &BTreeNode{leaf: true},
		degree: degree,
		size:   0,
	}
}

// Size returns the number of key-value pairs in the tree
func (t *BTree) Size() int {
	t.mu.RLock()
	defer t.mu.RUnlock()

	return t.size
}

// Get retrieves a value from the tree
func (t *BTree) Get(key string) (any, bool) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	return t.search(t.root, key)
}

// search searches for a key in the given node and its children
func (t *BTree) search(node *BTreeNode, key string) (any, bool) {
	i := 0
	for i < len(node.keys) && key > node.keys[i] {
		i++
	}

	if i < len(node.keys) && key == node.keys[i] {
		return node.values[i], true
	}

	if node.leaf {
		return nil, false
	}

	return t.search(node.children[i], key)
}

// Put adds or updates a key-value pair in the tree
func (t *BTree) Put(key string, value any) {
	t.mu.Lock()
	defer t.mu.Unlock()

	root := t.root

	// If the root is full, split it
	if len(root.keys) == 2*t.degree-1 {
		newRoot := &BTreeNode{leaf: false}
		newRoot.children = append(newRoot.children, root)
		t.splitChild(newRoot, 0)
		t.root = newRoot
		t.insertNonFull(newRoot, key, value)
	} else {
		t.insertNonFull(root, key, value)
	}

	t.size++
}

// splitChild splits a full child of a node
func (t *BTree) splitChild(parent *BTreeNode, index int) {
	degree := t.degree
	child := parent.children[index]

	// Create a new node to store the right half of the child's keys and values
	newChild := &BTreeNode{leaf: child.leaf}

	// Move the right half of the keys and values to the new node
	newChild.keys = append(newChild.keys, child.keys[degree:]...)
	newChild.values = append(newChild.values, child.values[degree:]...)

	// If the child is not a leaf, move the right half of its children too
	if !child.leaf {
		newChild.children = append(newChild.children, child.children[degree:]...)
	}

	// Truncate the child's keys, values, and children
	child.keys = child.keys[:degree-1]
	child.values = child.values[:degree-1]
	if !child.leaf {
		child.children = child.children[:degree]
	}

	// Insert the middle key and value into the parent
	parent.keys = append(parent.keys, "")
	parent.values = append(parent.values, nil)
	copy(parent.keys[index+1:], parent.keys[index:])
	copy(parent.values[index+1:], parent.values[index:])
	parent.keys[index] = child.keys[degree-1]
	parent.values[index] = child.values[degree-1]

	// Insert the new child into the parent's children
	parent.children = append(parent.children, nil)
	copy(parent.children[index+2:], parent.children[index+1:])
	parent.children[index+1] = newChild
}

// insertNonFull inserts a key-value pair into a non-full node
func (t *BTree) insertNonFull(node *BTreeNode, key string, value any) {
	i := len(node.keys) - 1

	if node.leaf {
		// Insert the key and value into the leaf node
		node.keys = append(node.keys, "")
		node.values = append(node.values, nil)

		for i >= 0 && key < node.keys[i] {
			node.keys[i+1] = node.keys[i]
			node.values[i+1] = node.values[i]
			i--
		}

		node.keys[i+1] = key
		node.values[i+1] = value
	} else {
		// Find the child to insert into
		for i >= 0 && key < node.keys[i] {
			i--
		}
		i++

		// If the child is full, split it
		if len(node.children[i].keys) == 2*t.degree-1 {
			t.splitChild(node, i)

			if key > node.keys[i] {
				i++
			}
		}

		t.insertNonFull(node.children[i], key, value)
	}
}

// Delete removes a key-value pair from the tree
func (t *BTree) Delete(key string) bool {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.root == nil {
		return false
	}

	result := t.delete(t.root, key)

	// If the root has no keys, make its first child the new root
	if len(t.root.keys) == 0 && !t.root.leaf {
		t.root = t.root.children[0]
	}

	if result {
		t.size--
	}

	return result
}

// delete removes a key-value pair from a node and its children
func (t *BTree) delete(node *BTreeNode, key string) bool {
	degree := t.degree

	// Find the position of the key
	i := 0
	for i < len(node.keys) && key > node.keys[i] {
		i++
	}

	// If the key is in this node
	if i < len(node.keys) && key == node.keys[i] {
		if node.leaf {
			// Remove the key and value from the leaf node
			node.keys = slices.Delete(node.keys, i, i+1)
			node.values = slices.Delete(node.values, i, i+1)
			return true
		}

		// Replace the key with its predecessor or successor
		if len(node.children[i].keys) >= degree {
			// Replace with predecessor
			predecessor, predecessorValue := t.findMax(node.children[i])
			node.keys[i] = predecessor
			node.values[i] = predecessorValue
			return t.delete(node.children[i], predecessor)
		} else if len(node.children[i+1].keys) >= degree {
			// Replace with successor
			successor, successorValue := t.findMin(node.children[i+1])
			node.keys[i] = successor
			node.values[i] = successorValue
			return t.delete(node.children[i+1], successor)
		}

		// Merge the child and its sibling
		t.merge(node, i)
		return t.delete(node.children[i], key)
	}

	// The key is not in this node
	if node.leaf {
		return false
	}

	// Check if the child has enough keys
	if len(node.children[i].keys) < degree {
		t.fill(node, i)
	}

	// If the last child has been merged, the key is in the last child
	if i > len(node.keys) {
		return t.delete(node.children[i-1], key)
	}

	return t.delete(node.children[i], key)
}

// findMax finds the maximum key and its value in a subtree
func (t *BTree) findMax(node *BTreeNode) (string, any) {
	if node.leaf {
		return node.keys[len(node.keys)-1], node.values[len(node.values)-1]
	}

	return t.findMax(node.children[len(node.children)-1])
}

// findMin finds the minimum key and its value in a subtree
func (t *BTree) findMin(node *BTreeNode) (string, any) {
	if node.leaf {
		return node.keys[0], node.values[0]
	}

	return t.findMin(node.children[0])
}

// fill ensures that a child has at least t.degree keys
func (t *BTree) fill(node *BTreeNode, index int) {
	degree := t.degree

	// Try to borrow from the left sibling
	if index > 0 && len(node.children[index-1].keys) >= degree {
		t.borrowFromPrev(node, index)
	} else if index < len(node.keys) && len(node.children[index+1].keys) >= degree {
		// Try to borrow from the right sibling
		t.borrowFromNext(node, index)
	} else {
		// Merge with a sibling
		if index < len(node.keys) {
			t.merge(node, index)
		} else {
			t.merge(node, index-1)
		}
	}
}

// borrowFromPrev borrows a key from the left sibling
func (t *BTree) borrowFromPrev(node *BTreeNode, index int) {
	child := node.children[index]
	sibling := node.children[index-1]

	// Make space for the new key
	child.keys = append([]string{""}, child.keys...)
	child.values = append([]any{nil}, child.values...)

	// If the child is not a leaf, move a child from the sibling
	if !child.leaf {
		child.children = append([]*BTreeNode{sibling.children[len(sibling.children)-1]}, child.children...)
		sibling.children = sibling.children[:len(sibling.children)-1]
	}

	// Move a key from the parent to the child
	child.keys[0] = node.keys[index-1]
	child.values[0] = node.values[index-1]

	// Move a key from the sibling to the parent
	node.keys[index-1] = sibling.keys[len(sibling.keys)-1]
	node.values[index-1] = sibling.values[len(sibling.values)-1]

	// Remove the key from the sibling
	sibling.keys = sibling.keys[:len(sibling.keys)-1]
	sibling.values = sibling.values[:len(sibling.values)-1]
}

// borrowFromNext borrows a key from the right sibling
func (t *BTree) borrowFromNext(node *BTreeNode, index int) {
	child := node.children[index]
	sibling := node.children[index+1]

	// Move a key from the parent to the child
	child.keys = append(child.keys, node.keys[index])
	child.values = append(child.values, node.values[index])

	// If the child is not a leaf, move a child from the sibling
	if !child.leaf {
		child.children = append(child.children, sibling.children[0])
		sibling.children = sibling.children[1:]
	}

	// Move a key from the sibling to the parent
	node.keys[index] = sibling.keys[0]
	node.values[index] = sibling.values[0]

	// Remove the key from the sibling
	sibling.keys = sibling.keys[1:]
	sibling.values = sibling.values[1:]
}

// merge merges a child with its next sibling
func (t *BTree) merge(node *BTreeNode, index int) {
	child := node.children[index]
	sibling := node.children[index+1]

	// Move a key from the parent to the child
	child.keys = append(child.keys, node.keys[index])
	child.values = append(child.values, node.values[index])

	// Move all keys and values from the sibling to the child
	child.keys = append(child.keys, sibling.keys...)
	child.values = append(child.values, sibling.values...)

	// If the child is not a leaf, move all children from the sibling to the child
	if !child.leaf {
		child.children = append(child.children, sibling.children...)
	}

	// Remove the key from the parent
	node.keys = slices.Delete(node.keys, index, index+1)
	node.values = slices.Delete(node.values, index, index+1)

	// Remove the sibling from the parent's children
	node.children = slices.Delete(node.children, index+1, index+2)
}

// ForEach executes the provided function for each key-value pair in the tree
func (t *BTree) ForEach(fn func(key string, value any) bool) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	if t.root == nil {
		return
	}

	t.forEach(t.root, fn)
}

// forEach executes the provided function for each key-value pair in a subtree
func (t *BTree) forEach(node *BTreeNode, fn func(key string, value any) bool) bool {
	if node == nil {
		return true
	}

	for i := range node.keys {
		if !node.leaf {
			if !t.forEach(node.children[i], fn) {
				return false
			}
		}

		if !fn(node.keys[i], node.values[i]) {
			return false
		}
	}

	if !node.leaf {
		return t.forEach(node.children[len(node.keys)], fn)
	}

	return true
}

// Clear removes all key-value pairs from the tree
func (t *BTree) Clear() {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.root = &BTreeNode{leaf: true}
	t.size = 0
}

// Keys returns all keys in the tree
func (t *BTree) Keys() []string {
	t.mu.RLock()
	defer t.mu.RUnlock()

	keys := make([]string, 0, t.size)

	t.ForEach(func(key string, _ any) bool {
		keys = append(keys, key)
		return true
	})

	return keys
}

// Values returns all values in the tree
func (t *BTree) Values() []any {
	t.mu.RLock()
	defer t.mu.RUnlock()

	values := make([]any, 0, t.size)

	t.ForEach(func(_ string, value any) bool {
		values = append(values, value)
		return true
	})

	return values
}
