// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.28.1
// 	protoc        v4.25.1
// source: proto/tunnel.proto

package proto

import (
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
	reflect "reflect"
	sync "sync"
)

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

type Tunnel_Type int32

const (
	Tunnel_STANDARD Tunnel_Type = 0
	Tunnel_REVERSE  Tunnel_Type = 1
)

// Enum value maps for Tunnel_Type.
var (
	Tunnel_Type_name = map[int32]string{
		0: "STANDARD",
		1: "REVERSE",
	}
	Tunnel_Type_value = map[string]int32{
		"STANDARD": 0,
		"REVERSE":  1,
	}
)

func (x Tunnel_Type) Enum() *Tunnel_Type {
	p := new(Tunnel_Type)
	*p = x
	return p
}

func (x Tunnel_Type) String() string {
	return protoimpl.X.EnumStringOf(x.Descriptor(), protoreflect.EnumNumber(x))
}

func (Tunnel_Type) Descriptor() protoreflect.EnumDescriptor {
	return file_proto_tunnel_proto_enumTypes[0].Descriptor()
}

func (Tunnel_Type) Type() protoreflect.EnumType {
	return &file_proto_tunnel_proto_enumTypes[0]
}

func (x Tunnel_Type) Number() protoreflect.EnumNumber {
	return protoreflect.EnumNumber(x)
}

// Deprecated: Use Tunnel_Type.Descriptor instead.
func (Tunnel_Type) EnumDescriptor() ([]byte, []int) {
	return file_proto_tunnel_proto_rawDescGZIP(), []int{0, 0}
}

type Tunnel struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Id       string      `protobuf:"bytes,1,opt,name=id,proto3" json:"id,omitempty"`
	Type     Tunnel_Type `protobuf:"varint,2,opt,name=type,proto3,enum=Tunnel_Type" json:"type,omitempty"`
	Enabled  bool        `protobuf:"varint,3,opt,name=enabled,proto3" json:"enabled,omitempty"`
	BindPort uint32      `protobuf:"varint,4,opt,name=BindPort,proto3" json:"BindPort,omitempty"`
	// Types that are assignable to Tunnel:
	//
	//	*Tunnel_StandardTunnel_
	//	*Tunnel_ReverseTunnel_
	Tunnel isTunnel_Tunnel `protobuf_oneof:"tunnel"`
}

func (x *Tunnel) Reset() {
	*x = Tunnel{}
	if protoimpl.UnsafeEnabled {
		mi := &file_proto_tunnel_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Tunnel) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Tunnel) ProtoMessage() {}

func (x *Tunnel) ProtoReflect() protoreflect.Message {
	mi := &file_proto_tunnel_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Tunnel.ProtoReflect.Descriptor instead.
func (*Tunnel) Descriptor() ([]byte, []int) {
	return file_proto_tunnel_proto_rawDescGZIP(), []int{0}
}

func (x *Tunnel) GetId() string {
	if x != nil {
		return x.Id
	}
	return ""
}

func (x *Tunnel) GetType() Tunnel_Type {
	if x != nil {
		return x.Type
	}
	return Tunnel_STANDARD
}

func (x *Tunnel) GetEnabled() bool {
	if x != nil {
		return x.Enabled
	}
	return false
}

func (x *Tunnel) GetBindPort() uint32 {
	if x != nil {
		return x.BindPort
	}
	return 0
}

func (m *Tunnel) GetTunnel() isTunnel_Tunnel {
	if m != nil {
		return m.Tunnel
	}
	return nil
}

func (x *Tunnel) GetStandardTunnel() *Tunnel_StandardTunnel {
	if x, ok := x.GetTunnel().(*Tunnel_StandardTunnel_); ok {
		return x.StandardTunnel
	}
	return nil
}

func (x *Tunnel) GetReverseTunnel() *Tunnel_ReverseTunnel {
	if x, ok := x.GetTunnel().(*Tunnel_ReverseTunnel_); ok {
		return x.ReverseTunnel
	}
	return nil
}

type isTunnel_Tunnel interface {
	isTunnel_Tunnel()
}

type Tunnel_StandardTunnel_ struct {
	StandardTunnel *Tunnel_StandardTunnel `protobuf:"bytes,6,opt,name=standard_tunnel,json=standardTunnel,proto3,oneof"`
}

type Tunnel_ReverseTunnel_ struct {
	ReverseTunnel *Tunnel_ReverseTunnel `protobuf:"bytes,5,opt,name=reverse_tunnel,json=reverseTunnel,proto3,oneof"`
}

func (*Tunnel_StandardTunnel_) isTunnel_Tunnel() {}

func (*Tunnel_ReverseTunnel_) isTunnel_Tunnel() {}

type GetTunnelRequest struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Id string `protobuf:"bytes,1,opt,name=id,proto3" json:"id,omitempty"`
}

func (x *GetTunnelRequest) Reset() {
	*x = GetTunnelRequest{}
	if protoimpl.UnsafeEnabled {
		mi := &file_proto_tunnel_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *GetTunnelRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*GetTunnelRequest) ProtoMessage() {}

func (x *GetTunnelRequest) ProtoReflect() protoreflect.Message {
	mi := &file_proto_tunnel_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use GetTunnelRequest.ProtoReflect.Descriptor instead.
func (*GetTunnelRequest) Descriptor() ([]byte, []int) {
	return file_proto_tunnel_proto_rawDescGZIP(), []int{1}
}

func (x *GetTunnelRequest) GetId() string {
	if x != nil {
		return x.Id
	}
	return ""
}

type Tunnel_StandardTunnel struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	SshHost     string `protobuf:"bytes,1,opt,name=SshHost,proto3" json:"SshHost,omitempty"`
	SshPort     uint32 `protobuf:"varint,2,opt,name=SshPort,proto3" json:"SshPort,omitempty"`
	ServiceHost string `protobuf:"bytes,3,opt,name=ServiceHost,proto3" json:"ServiceHost,omitempty"`
	ServicePort string `protobuf:"bytes,4,opt,name=ServicePort,proto3" json:"ServicePort,omitempty"`
}

func (x *Tunnel_StandardTunnel) Reset() {
	*x = Tunnel_StandardTunnel{}
	if protoimpl.UnsafeEnabled {
		mi := &file_proto_tunnel_proto_msgTypes[2]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Tunnel_StandardTunnel) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Tunnel_StandardTunnel) ProtoMessage() {}

func (x *Tunnel_StandardTunnel) ProtoReflect() protoreflect.Message {
	mi := &file_proto_tunnel_proto_msgTypes[2]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Tunnel_StandardTunnel.ProtoReflect.Descriptor instead.
func (*Tunnel_StandardTunnel) Descriptor() ([]byte, []int) {
	return file_proto_tunnel_proto_rawDescGZIP(), []int{0, 0}
}

func (x *Tunnel_StandardTunnel) GetSshHost() string {
	if x != nil {
		return x.SshHost
	}
	return ""
}

func (x *Tunnel_StandardTunnel) GetSshPort() uint32 {
	if x != nil {
		return x.SshPort
	}
	return 0
}

func (x *Tunnel_StandardTunnel) GetServiceHost() string {
	if x != nil {
		return x.ServiceHost
	}
	return ""
}

func (x *Tunnel_StandardTunnel) GetServicePort() string {
	if x != nil {
		return x.ServicePort
	}
	return ""
}

type Tunnel_ReverseTunnel struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields
}

func (x *Tunnel_ReverseTunnel) Reset() {
	*x = Tunnel_ReverseTunnel{}
	if protoimpl.UnsafeEnabled {
		mi := &file_proto_tunnel_proto_msgTypes[3]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Tunnel_ReverseTunnel) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Tunnel_ReverseTunnel) ProtoMessage() {}

func (x *Tunnel_ReverseTunnel) ProtoReflect() protoreflect.Message {
	mi := &file_proto_tunnel_proto_msgTypes[3]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Tunnel_ReverseTunnel.ProtoReflect.Descriptor instead.
func (*Tunnel_ReverseTunnel) Descriptor() ([]byte, []int) {
	return file_proto_tunnel_proto_rawDescGZIP(), []int{0, 1}
}

var File_proto_tunnel_proto protoreflect.FileDescriptor

var file_proto_tunnel_proto_rawDesc = []byte{
	0x0a, 0x12, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x2f, 0x74, 0x75, 0x6e, 0x6e, 0x65, 0x6c, 0x2e, 0x70,
	0x72, 0x6f, 0x74, 0x6f, 0x22, 0xbc, 0x03, 0x0a, 0x06, 0x54, 0x75, 0x6e, 0x6e, 0x65, 0x6c, 0x12,
	0x0e, 0x0a, 0x02, 0x69, 0x64, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x02, 0x69, 0x64, 0x12,
	0x20, 0x0a, 0x04, 0x74, 0x79, 0x70, 0x65, 0x18, 0x02, 0x20, 0x01, 0x28, 0x0e, 0x32, 0x0c, 0x2e,
	0x54, 0x75, 0x6e, 0x6e, 0x65, 0x6c, 0x2e, 0x54, 0x79, 0x70, 0x65, 0x52, 0x04, 0x74, 0x79, 0x70,
	0x65, 0x12, 0x18, 0x0a, 0x07, 0x65, 0x6e, 0x61, 0x62, 0x6c, 0x65, 0x64, 0x18, 0x03, 0x20, 0x01,
	0x28, 0x08, 0x52, 0x07, 0x65, 0x6e, 0x61, 0x62, 0x6c, 0x65, 0x64, 0x12, 0x1a, 0x0a, 0x08, 0x42,
	0x69, 0x6e, 0x64, 0x50, 0x6f, 0x72, 0x74, 0x18, 0x04, 0x20, 0x01, 0x28, 0x0d, 0x52, 0x08, 0x42,
	0x69, 0x6e, 0x64, 0x50, 0x6f, 0x72, 0x74, 0x12, 0x41, 0x0a, 0x0f, 0x73, 0x74, 0x61, 0x6e, 0x64,
	0x61, 0x72, 0x64, 0x5f, 0x74, 0x75, 0x6e, 0x6e, 0x65, 0x6c, 0x18, 0x06, 0x20, 0x01, 0x28, 0x0b,
	0x32, 0x16, 0x2e, 0x54, 0x75, 0x6e, 0x6e, 0x65, 0x6c, 0x2e, 0x53, 0x74, 0x61, 0x6e, 0x64, 0x61,
	0x72, 0x64, 0x54, 0x75, 0x6e, 0x6e, 0x65, 0x6c, 0x48, 0x00, 0x52, 0x0e, 0x73, 0x74, 0x61, 0x6e,
	0x64, 0x61, 0x72, 0x64, 0x54, 0x75, 0x6e, 0x6e, 0x65, 0x6c, 0x12, 0x3e, 0x0a, 0x0e, 0x72, 0x65,
	0x76, 0x65, 0x72, 0x73, 0x65, 0x5f, 0x74, 0x75, 0x6e, 0x6e, 0x65, 0x6c, 0x18, 0x05, 0x20, 0x01,
	0x28, 0x0b, 0x32, 0x15, 0x2e, 0x54, 0x75, 0x6e, 0x6e, 0x65, 0x6c, 0x2e, 0x52, 0x65, 0x76, 0x65,
	0x72, 0x73, 0x65, 0x54, 0x75, 0x6e, 0x6e, 0x65, 0x6c, 0x48, 0x00, 0x52, 0x0d, 0x72, 0x65, 0x76,
	0x65, 0x72, 0x73, 0x65, 0x54, 0x75, 0x6e, 0x6e, 0x65, 0x6c, 0x1a, 0x88, 0x01, 0x0a, 0x0e, 0x53,
	0x74, 0x61, 0x6e, 0x64, 0x61, 0x72, 0x64, 0x54, 0x75, 0x6e, 0x6e, 0x65, 0x6c, 0x12, 0x18, 0x0a,
	0x07, 0x53, 0x73, 0x68, 0x48, 0x6f, 0x73, 0x74, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x07,
	0x53, 0x73, 0x68, 0x48, 0x6f, 0x73, 0x74, 0x12, 0x18, 0x0a, 0x07, 0x53, 0x73, 0x68, 0x50, 0x6f,
	0x72, 0x74, 0x18, 0x02, 0x20, 0x01, 0x28, 0x0d, 0x52, 0x07, 0x53, 0x73, 0x68, 0x50, 0x6f, 0x72,
	0x74, 0x12, 0x20, 0x0a, 0x0b, 0x53, 0x65, 0x72, 0x76, 0x69, 0x63, 0x65, 0x48, 0x6f, 0x73, 0x74,
	0x18, 0x03, 0x20, 0x01, 0x28, 0x09, 0x52, 0x0b, 0x53, 0x65, 0x72, 0x76, 0x69, 0x63, 0x65, 0x48,
	0x6f, 0x73, 0x74, 0x12, 0x20, 0x0a, 0x0b, 0x53, 0x65, 0x72, 0x76, 0x69, 0x63, 0x65, 0x50, 0x6f,
	0x72, 0x74, 0x18, 0x04, 0x20, 0x01, 0x28, 0x09, 0x52, 0x0b, 0x53, 0x65, 0x72, 0x76, 0x69, 0x63,
	0x65, 0x50, 0x6f, 0x72, 0x74, 0x1a, 0x0f, 0x0a, 0x0d, 0x52, 0x65, 0x76, 0x65, 0x72, 0x73, 0x65,
	0x54, 0x75, 0x6e, 0x6e, 0x65, 0x6c, 0x22, 0x21, 0x0a, 0x04, 0x54, 0x79, 0x70, 0x65, 0x12, 0x0c,
	0x0a, 0x08, 0x53, 0x54, 0x41, 0x4e, 0x44, 0x41, 0x52, 0x44, 0x10, 0x00, 0x12, 0x0b, 0x0a, 0x07,
	0x52, 0x45, 0x56, 0x45, 0x52, 0x53, 0x45, 0x10, 0x01, 0x42, 0x08, 0x0a, 0x06, 0x74, 0x75, 0x6e,
	0x6e, 0x65, 0x6c, 0x22, 0x22, 0x0a, 0x10, 0x47, 0x65, 0x74, 0x54, 0x75, 0x6e, 0x6e, 0x65, 0x6c,
	0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x12, 0x0e, 0x0a, 0x02, 0x69, 0x64, 0x18, 0x01, 0x20,
	0x01, 0x28, 0x09, 0x52, 0x02, 0x69, 0x64, 0x32, 0x32, 0x0a, 0x07, 0x50, 0x61, 0x73, 0x73, 0x61,
	0x67, 0x65, 0x12, 0x27, 0x0a, 0x09, 0x47, 0x65, 0x74, 0x54, 0x75, 0x6e, 0x6e, 0x65, 0x6c, 0x12,
	0x11, 0x2e, 0x47, 0x65, 0x74, 0x54, 0x75, 0x6e, 0x6e, 0x65, 0x6c, 0x52, 0x65, 0x71, 0x75, 0x65,
	0x73, 0x74, 0x1a, 0x07, 0x2e, 0x54, 0x75, 0x6e, 0x6e, 0x65, 0x6c, 0x42, 0x2d, 0x5a, 0x2b, 0x67,
	0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f, 0x68, 0x69, 0x67, 0x68, 0x74, 0x6f,
	0x75, 0x63, 0x68, 0x69, 0x6f, 0x2f, 0x70, 0x61, 0x73, 0x73, 0x61, 0x67, 0x65, 0x2f, 0x74, 0x75,
	0x6e, 0x6e, 0x65, 0x6c, 0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x06, 0x70, 0x72, 0x6f, 0x74,
	0x6f, 0x33,
}

var (
	file_proto_tunnel_proto_rawDescOnce sync.Once
	file_proto_tunnel_proto_rawDescData = file_proto_tunnel_proto_rawDesc
)

func file_proto_tunnel_proto_rawDescGZIP() []byte {
	file_proto_tunnel_proto_rawDescOnce.Do(func() {
		file_proto_tunnel_proto_rawDescData = protoimpl.X.CompressGZIP(file_proto_tunnel_proto_rawDescData)
	})
	return file_proto_tunnel_proto_rawDescData
}

var file_proto_tunnel_proto_enumTypes = make([]protoimpl.EnumInfo, 1)
var file_proto_tunnel_proto_msgTypes = make([]protoimpl.MessageInfo, 4)
var file_proto_tunnel_proto_goTypes = []interface{}{
	(Tunnel_Type)(0),              // 0: Tunnel.Type
	(*Tunnel)(nil),                // 1: Tunnel
	(*GetTunnelRequest)(nil),      // 2: GetTunnelRequest
	(*Tunnel_StandardTunnel)(nil), // 3: Tunnel.StandardTunnel
	(*Tunnel_ReverseTunnel)(nil),  // 4: Tunnel.ReverseTunnel
}
var file_proto_tunnel_proto_depIdxs = []int32{
	0, // 0: Tunnel.type:type_name -> Tunnel.Type
	3, // 1: Tunnel.standard_tunnel:type_name -> Tunnel.StandardTunnel
	4, // 2: Tunnel.reverse_tunnel:type_name -> Tunnel.ReverseTunnel
	2, // 3: Passage.GetTunnel:input_type -> GetTunnelRequest
	1, // 4: Passage.GetTunnel:output_type -> Tunnel
	4, // [4:5] is the sub-list for method output_type
	3, // [3:4] is the sub-list for method input_type
	3, // [3:3] is the sub-list for extension type_name
	3, // [3:3] is the sub-list for extension extendee
	0, // [0:3] is the sub-list for field type_name
}

func init() { file_proto_tunnel_proto_init() }
func file_proto_tunnel_proto_init() {
	if File_proto_tunnel_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_proto_tunnel_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*Tunnel); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_proto_tunnel_proto_msgTypes[1].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*GetTunnelRequest); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_proto_tunnel_proto_msgTypes[2].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*Tunnel_StandardTunnel); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_proto_tunnel_proto_msgTypes[3].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*Tunnel_ReverseTunnel); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
	}
	file_proto_tunnel_proto_msgTypes[0].OneofWrappers = []interface{}{
		(*Tunnel_StandardTunnel_)(nil),
		(*Tunnel_ReverseTunnel_)(nil),
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: file_proto_tunnel_proto_rawDesc,
			NumEnums:      1,
			NumMessages:   4,
			NumExtensions: 0,
			NumServices:   1,
		},
		GoTypes:           file_proto_tunnel_proto_goTypes,
		DependencyIndexes: file_proto_tunnel_proto_depIdxs,
		EnumInfos:         file_proto_tunnel_proto_enumTypes,
		MessageInfos:      file_proto_tunnel_proto_msgTypes,
	}.Build()
	File_proto_tunnel_proto = out.File
	file_proto_tunnel_proto_rawDesc = nil
	file_proto_tunnel_proto_goTypes = nil
	file_proto_tunnel_proto_depIdxs = nil
}
