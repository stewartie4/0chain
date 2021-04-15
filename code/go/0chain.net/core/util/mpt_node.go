package util

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/ioutil"

	"0chain.net/core/encryption"
)

const (
	NodeTypeValueNode     = 1
	NodeTypeLeafNode      = 2
	NodeTypeFullNode      = 4
	NodeTypeExtensionNode = 8
	NodeTypesAll          = NodeTypeValueNode | NodeTypeLeafNode | NodeTypeFullNode | NodeTypeExtensionNode
)

//Separator - used to separate fields when creating data array to hash
const Separator = ':'

//ErrInvalidEncoding - error to indicate invalid encoding
var ErrInvalidEncoding = errors.New("invalid node encoding")

//PathElements - all the bytes that can be used as path elements as ascii characters
var PathElements = []byte("0123456789abcdef")

/*Node - a node interface */
type Node interface {
	Clone() Node
	GetNodeType() byte
	SecureSerializableValueI
	OriginTrackerI
	GetOriginTracker() OriginTrackerI
	SetOriginTracker(ot OriginTrackerI)
}

//OriginTrackerNode - a node that implements origin tracking
type OriginTrackerNode struct {
	OriginTracker OriginTrackerI `json:"o,omitempty"`
}

//NewOriginTrackerNode - create a new origin tracker node
func NewOriginTrackerNode() *OriginTrackerNode {
	otn := &OriginTrackerNode{}
	otn.OriginTracker = &OriginTracker{}
	return otn
}

//Clone - clone the given origin tracker node
func (otn *OriginTrackerNode) Clone() *OriginTrackerNode {
	clone := NewOriginTrackerNode()
	clone.OriginTracker.SetOrigin(otn.GetOrigin())
	clone.OriginTracker.SetVersion(otn.GetVersion())
	return clone
}

//SetOriginTracker - implement interface
func (otn *OriginTrackerNode) SetOriginTracker(ot OriginTrackerI) {
	otn.OriginTracker = ot
}

//SetOrigin - implement interface
func (otn *OriginTrackerNode) SetOrigin(origin Sequence) {
	otn.OriginTracker.SetOrigin(origin)
}

//GetOrigin - implement interface
func (otn *OriginTrackerNode) GetOrigin() Sequence {
	return otn.OriginTracker.GetOrigin()
}

//SetVersion - implement interface
func (otn *OriginTrackerNode) SetVersion(version Sequence) {
	otn.OriginTracker.SetVersion(version)
}

//GetVersion - implement interface
func (otn *OriginTrackerNode) GetVersion() Sequence {
	return otn.OriginTracker.GetVersion()
}

//Write - implement interface
func (otn *OriginTrackerNode) Write(w io.Writer) error {
	return otn.OriginTracker.Write(w)
}

//Read - implement interface
func (otn *OriginTrackerNode) Read(r io.Reader) error {
	return otn.OriginTracker.Read(r)
}

//GetOriginTracker - implement interface
func (otn *OriginTrackerNode) GetOriginTracker() OriginTrackerI {
	return otn.OriginTracker
}

type secureCacheableNode interface {
	Node
	encode() (enc []byte)
	hash(enc []byte) (hash []byte)
}

type nodeCache struct {
	cached           bool
	_hash            string
	_hashBytes       []byte
	_valueBytes      []byte
	_nodeBytes       []byte
	_nodePrefixBytes []byte
}

func (nc *nodeCache) clear()             { *nc = nodeCache{} }
func (nc *nodeCache) hash() string       { return nc._hash }
func (nc *nodeCache) hashBytes() []byte  { return nc._hashBytes }
func (nc *nodeCache) nodeBytes() []byte  { return nc._nodeBytes }
func (nc *nodeCache) valueBytes() []byte { return nc._valueBytes }

func (nc *nodeCache) cache(n secureCacheableNode) *nodeCache {
	encodeNode := false
	if n == nil {
		nc.clear()
		return nc
	}
	if !nc.cached {
		nc._valueBytes = n.encode()
		nc._hashBytes = n.hash(nc._valueBytes)
		nc._hash = ToHex(nc._hashBytes)
		nc.cached = true
		encodeNode = true
	}
	if ot, ok := n.GetOriginTracker().(*OriginTracker); ok && !ot.cached {
		buf := bytes.NewBuffer(nil)
		writeNodePrefix(buf, n)
		nc._nodePrefixBytes = buf.Bytes() // set nodePrefix
		ot.cached = true
		encodeNode = true
	}
	if encodeNode && len(nc._nodePrefixBytes) > 0 {
		bufSize := len(nc._nodePrefixBytes) + len(nc._valueBytes)
		nc._nodeBytes = make([]byte, bufSize)
		copy(nc._nodeBytes[:len(nc._nodePrefixBytes)], nc._nodePrefixBytes)
		copy(nc._nodeBytes[len(nc._nodePrefixBytes):], nc._valueBytes)
		nc._valueBytes = nc._nodeBytes[len(nc._nodePrefixBytes):] // reuse part of the slice to avoid replication
	}
	return nc
}

/*ValueNode - any node that holds a value should implement this */
type ValueNode struct {
	Value              Serializable `json:"v"`
	*OriginTrackerNode `json:"o,omitempty"`
	nc                 nodeCache
}

//NewValueNode - create a new value node
func NewValueNode() *ValueNode {
	vn := &ValueNode{}
	vn.OriginTrackerNode = NewOriginTrackerNode()
	return vn
}

/*Clone - implement interface */
func (vn *ValueNode) Clone() Node {
	clone := NewValueNode()
	clone.OriginTrackerNode = vn.OriginTrackerNode.Clone()
	clone.SetValue(vn.GetValue())
	clone.nc = vn.nc
	return clone
}

/*GetNodeType - implement interface */
func (vn *ValueNode) GetNodeType() byte {
	return NodeTypeValueNode
}

/*GetHash - implements SecureSerializableValue interface */
func (vn *ValueNode) GetHash() string {
	return vn.nc.cache(vn).hash()
}

/*GetHashBytes - implement SecureSerializableValue interface */
func (vn *ValueNode) GetHashBytes() []byte {
	return vn.nc.cache(vn).hashBytes()
}

func (vn *ValueNode) hash(encodedValue []byte) []byte {
	if len(encodedValue) > 0 {
		return encryption.RawHash(encodedValue)
	}
	return nil
}

func (vn *ValueNode) encode() []byte {
	if vn.Value != nil {
		return vn.Value.Encode()
	}
	return nil
}

/*HasValue - check if the value stored is empty */
func (vn *ValueNode) HasValue() bool {
	return len(vn.nc.cache(vn).valueBytes()) > 0
}

/*Encode - overwrite interface method */
func (vn *ValueNode) Encode() []byte {
	return vn.nc.cache(vn).nodeBytes()
}

/*Decode - overwrite interface method */
func (vn *ValueNode) Decode(buf []byte) error {
	ssv := &SecureSerializableValue{}
	err := ssv.Decode(buf)
	if err != nil {
		return err
	}
	vn.SetValue(ssv)
	return nil
}

/*GetValue - get the value store in this node */
func (vn *ValueNode) GetValue() Serializable {
	return vn.Value
}

/*SetValue - set the value stored in this node */
func (vn *ValueNode) SetValue(value Serializable) {
	vn.Value = value
	vn.nc.clear()
}

/*LeafNode - a node that represents the leaf that contains a value and an optional path */
type LeafNode struct {
	Path               Path       `json:"p,omitempty"`
	Value              *ValueNode `json:"v"`
	*OriginTrackerNode `json:"o"`
	nc                 nodeCache
}

/*NewLeafNode - create a new leaf node */
func NewLeafNode(path Path, origin Sequence, value Serializable) *LeafNode {
	ln := &LeafNode{}
	ln.OriginTrackerNode = NewOriginTrackerNode()
	ln.SetPath(path)
	ln.SetValue(value)
	ln.SetOrigin(origin)
	return ln
}

/*Clone - implement interface */
func (ln *LeafNode) Clone() Node {
	clone := &LeafNode{}
	clone.OriginTrackerNode = ln.OriginTrackerNode.Clone()
	clone.SetPath(ln.Path) // path will never be updated inplace and so ok
	clone.SetValue(ln.GetValue())
	clone.nc = ln.nc
	return clone
}

/*GetHash - implements SecureSerializableValue interface */
func (ln *LeafNode) GetHash() string {
	return ln.nc.cache(ln).hash()
}

/*GetHashBytes - implement interface */
func (ln *LeafNode) GetHashBytes() []byte {
	return ln.nc.cache(ln).hashBytes()
}

/*GetNodeType - implement interface */
func (ln *LeafNode) GetNodeType() byte {
	return NodeTypeLeafNode
}

func (ln *LeafNode) hash(encodedValue []byte) []byte {
	if len(encodedValue) > 0 {
		buf := bytes.NewBuffer(nil)
		binary.Write(buf, binary.LittleEndian, ln.GetOrigin()) // Why is this done?
		buf.Write(encodedValue)
		return encryption.RawHash(buf.Bytes())
	}
	return nil
}

func (ln *LeafNode) encode() []byte {
	buf := bytes.NewBuffer(nil)
	if len(ln.Path) > 0 {
		buf.Write(ln.Path)
	}
	buf.WriteByte(Separator)
	if ln.HasValue() {
		buf.Write(ln.GetValue().Encode())
	}
	return buf.Bytes()
}

/*Encode - implement interface */
func (ln *LeafNode) Encode() []byte {
	return ln.nc.cache(ln).nodeBytes()
}

/*Decode - implement interface */
func (ln *LeafNode) Decode(buf []byte) error {
	idx := bytes.IndexByte(buf, Separator)
	if idx < 0 {
		return ErrInvalidEncoding
	}
	ln.SetPath(buf[:idx])
	buf = buf[idx+1:]
	var v Serializable
	if len(buf) > 0 {
		vn := NewValueNode()
		vn.Decode(buf)
		v = vn.GetValue()
	}
	ln.SetValue(v)
	return nil
}

/*HasValue - implement interface */
func (ln *LeafNode) HasValue() bool {
	return ln.Value != nil && ln.Value.HasValue()
}

/*GetValue - implement interface */
func (ln *LeafNode) GetValue() Serializable {
	if ln.HasValue() {
		return ln.Value.GetValue()
	}
	return nil
}

/*SetValue - implement interface */
func (ln *LeafNode) SetValue(value Serializable) {
	if ln.Value == nil {
		ln.Value = NewValueNode()
	}
	ln.Value.SetValue(value)
	ln.nc.clear()
}

func (ln *LeafNode) SetPath(path Path) {
	ln.Path = path
	ln.nc.clear()
}

/*FullNode - a branch node that can contain 16 children and a value */
type FullNode struct {
	Children           [16][]byte `json:"c"`
	Value              *ValueNode `json:"v,omitempty"` // This may not be needed as our path is fixed in size
	*OriginTrackerNode `json:"o,omitempty"`
	nc                 nodeCache
}

/*NewFullNode - create a new full node */
func NewFullNode(value Serializable) *FullNode {
	fn := &FullNode{}
	fn.OriginTrackerNode = NewOriginTrackerNode()
	fn.SetValue(value)
	return fn
}

/*Clone - implement interface */
func (fn *FullNode) Clone() Node {
	clone := &FullNode{}
	clone.OriginTrackerNode = fn.OriginTrackerNode.Clone()
	for idx, ckey := range fn.Children {
		clone.Children[idx] = ckey // ckey will never be updated inplace and so ok
	}
	clone.SetValue(fn.GetValue())
	clone.nc = fn.nc
	return clone
}

/*GetHash - implements SecureSerializableValue interface */
func (fn *FullNode) GetHash() string {
	return fn.nc.cache(fn).hash()
}

/*GetHashBytes - implement interface */
func (fn *FullNode) GetHashBytes() []byte {
	return fn.nc.cache(fn).hashBytes()
}

func (fn *FullNode) hash(encodedValue []byte) []byte {
	if len(encodedValue) > 0 {
		return encryption.RawHash(encodedValue)
	}
	return nil
}

func (fn *FullNode) encode() []byte {
	buf := bytes.NewBuffer(nil)
	for i := byte(0); i < 16; i++ {
		child := fn.GetChild(fn.indexToByte(i))
		if child != nil {
			buf.Write([]byte(ToHex(child)))
		}
		buf.WriteByte(Separator)
	}
	if fn.HasValue() {
		buf.Write(fn.GetValue().Encode())
	}
	return buf.Bytes()
}

/*Encode - implement interface */
func (fn *FullNode) Encode() []byte {
	return fn.nc.cache(fn).nodeBytes()
}

/*Decode - implement interface */
func (fn *FullNode) Decode(buf []byte) error {
	for i := byte(0); i < 16; i++ {
		idx := bytes.IndexByte(buf, Separator)
		if idx < 0 {
			return ErrInvalidEncoding
		}
		if idx > 0 {
			key := make([]byte, 32)
			_, err := hex.Decode(key, buf[:idx])
			if err != nil {
				return err
			}
			fn.PutChild(fn.indexToByte(i), key)
		}
		buf = buf[idx+1:]
	}
	var v Serializable
	if len(buf) > 0 {
		vn := NewValueNode()
		vn.Decode(buf)
		v = vn.GetValue()
	}
	fn.SetValue(v)
	return nil
}

/*GetNodeType - implement interface */
func (fn *FullNode) GetNodeType() byte {
	return NodeTypeFullNode
}

func (fn *FullNode) index(c byte) byte {
	if c >= 48 && c <= 57 {
		return c - 48
	}
	if c >= 97 && c <= 102 {
		return 10 + c - 97
	}
	if c >= 65 && c <= 70 {
		return 10 + c - 65
	}
	panic("Invalid byte for index in Patricia Merkle Trie")
}

func (fn *FullNode) indexToByte(idx byte) byte {
	if idx < 10 {
		return 48 + idx
	}
	return 97 + (idx - 10)
}

/*GetNumChildren - get the number of children in this node */
func (fn *FullNode) GetNumChildren() byte {
	var count byte
	for _, child := range fn.Children {
		if child != nil {
			count++
		}
	}
	return count
}

/*GetChild - get the child at the given hex index */
func (fn *FullNode) GetChild(hex byte) []byte {
	return fn.Children[fn.index(hex)]
}

/*PutChild - put the child at the given hex index */
func (fn *FullNode) PutChild(hex byte, child []byte) {
	fn.Children[fn.index(hex)] = child
	fn.nc.clear()
}

/*HasValue - implement interface */
func (fn *FullNode) HasValue() bool {
	return fn.Value != nil && fn.Value.HasValue()
}

/*GetValue - implement interface */
func (fn *FullNode) GetValue() Serializable {
	if fn.HasValue() {
		return fn.Value.GetValue()
	}
	return nil
}

/*SetValue - implement interface */
func (fn *FullNode) SetValue(value Serializable) {
	if fn.Value == nil {
		fn.Value = NewValueNode()
	}
	fn.Value.SetValue(value)
	fn.nc.clear()
}

/*ExtensionNode - a multi-char length path along which there are no branches, at the end of this path there should be full node */
type ExtensionNode struct {
	Path               Path `json:"p"`
	NodeKey            Key  `json:"k"`
	*OriginTrackerNode `json:"o,omitempty"`
	nc                 nodeCache
}

/*NewExtensionNode - create a new extension node */
func NewExtensionNode(path Path, key Key) *ExtensionNode {
	en := &ExtensionNode{}
	en.OriginTrackerNode = NewOriginTrackerNode()
	en.SetPath(path)
	en.SetNodeKey(key)
	return en
}

/*Clone - implement interface */
func (en *ExtensionNode) Clone() Node {
	clone := &ExtensionNode{}
	clone.OriginTrackerNode = en.OriginTrackerNode.Clone()
	clone.SetPath(en.Path)
	clone.SetNodeKey(en.NodeKey)
	clone.nc = en.nc
	return clone
}

/*GetHash - implements SecureSerializableValue interface */
func (en *ExtensionNode) GetHash() string {
	return en.nc.cache(en).hash()
}

/*GetHashBytes - implement interface */
func (en *ExtensionNode) GetHashBytes() []byte {
	return en.nc.cache(en).hashBytes()
}

/*GetNodeType - implement interface */
func (en *ExtensionNode) GetNodeType() byte {
	return NodeTypeExtensionNode
}

func (en *ExtensionNode) hash(encodedValue []byte) []byte {
	if len(encodedValue) > 0 {
		return encryption.RawHash(encodedValue)
	}
	return nil
}

func (en *ExtensionNode) encode() []byte {
	buf := bytes.NewBuffer(make([]byte, 0, len(en.Path)+1+len(en.NodeKey)))
	buf.Write(en.Path)
	buf.WriteByte(Separator)
	buf.Write(en.NodeKey)
	return buf.Bytes()
}

/*Encode - implement interface */
func (en *ExtensionNode) Encode() []byte {
	return en.nc.cache(en).nodeBytes()
}

/*Decode - implement interface */
func (en *ExtensionNode) Decode(buf []byte) (err error) {
	idx := bytes.IndexByte(buf, Separator)
	if idx < 0 {
		return ErrInvalidEncoding
	}
	en.SetPath(buf[:idx])
	en.SetNodeKey(buf[idx+1:])
	return
}

func (en *ExtensionNode) SetPath(path Path) {
	en.Path = path
	en.nc.clear()
}

func (en *ExtensionNode) SetNodeKey(key Key) {
	en.NodeKey = key
	en.nc.clear()
}

/*GetValueNode - get the value node associated with this node*/
func GetValueNode(node Node) *ValueNode {
	if node == nil {
		return nil
	}
	switch nodeImpl := node.(type) {
	case *ValueNode:
		return nodeImpl
	case *LeafNode:
		return nodeImpl.Value
	case *FullNode:
		return nodeImpl.Value
	default:
		return nil
	}
}

/*GetSerializationPrefix - get the serialization prefix */
func GetSerializationPrefix(node Node) byte {
	switch node.(type) {
	case *ValueNode:
		return NodeTypeValueNode
	case *LeafNode:
		return NodeTypeLeafNode
	case *FullNode:
		return NodeTypeFullNode
	case *ExtensionNode:
		return NodeTypeExtensionNode
	default:
		panic("uknown node type")
	}
}

/*IncludesNodeType - checks if the given node type is one of the node types in the mask */
func IncludesNodeType(nodeTypes byte, nodeType byte) bool {
	return (nodeTypes & nodeType) == nodeType
}

/*CreateNode - create a node based on the serialization prefix */
func CreateNode(r io.Reader) (Node, error) {
	buf := []byte{0}
	n, err := r.Read(buf)
	if err != nil {
		return nil, err
	}
	if n == 0 {
		return nil, ErrInvalidEncoding
	}
	code := buf[0]
	var node Node
	switch code & NodeTypesAll {
	case NodeTypeValueNode:
		node = NewValueNode()
	case NodeTypeLeafNode:
		node = NewLeafNode(nil, Sequence(0), nil)
	case NodeTypeFullNode:
		node = NewFullNode(nil)
	case NodeTypeExtensionNode:
		node = NewExtensionNode(nil, nil)
	default:
		panic(fmt.Sprintf("unkown node type: %v", code))
	}
	var ot OriginTracker
	ot.Read(r)
	node.SetOriginTracker(&ot)
	buf, err = ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}
	err = node.Decode(buf)
	return node, err
}

func writeNodePrefix(w io.Writer, node Node) error {
	_, err := w.Write([]byte{GetSerializationPrefix(node)})
	if err != nil {
		return err
	}
	node.GetOriginTracker().Write(w)
	return nil
}
