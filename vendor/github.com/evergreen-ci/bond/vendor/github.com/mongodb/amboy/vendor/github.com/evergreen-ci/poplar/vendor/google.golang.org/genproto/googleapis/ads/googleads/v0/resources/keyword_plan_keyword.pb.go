// Code generated by protoc-gen-go. DO NOT EDIT.
// source: google/ads/googleads/v0/resources/keyword_plan_keyword.proto

package resources // import "google.golang.org/genproto/googleapis/ads/googleads/v0/resources"

import proto "github.com/golang/protobuf/proto"
import fmt "fmt"
import math "math"
import wrappers "github.com/golang/protobuf/ptypes/wrappers"
import enums "google.golang.org/genproto/googleapis/ads/googleads/v0/enums"

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

// This is a compile-time assertion to ensure that this generated file
// is compatible with the proto package it is being compiled against.
// A compilation error at this line likely means your copy of the
// proto package needs to be updated.
const _ = proto.ProtoPackageIsVersion2 // please upgrade the proto package

// A Keyword Plan ad group keyword.
// Max number of keyword plan keywords per plan: 2500.
type KeywordPlanKeyword struct {
	// The resource name of the Keyword Plan ad group keyword.
	// KeywordPlanKeyword resource names have the form:
	//
	// `customers/{customer_id}/keywordPlanKeywords/{kp_ad_group_keyword_id}`
	ResourceName string `protobuf:"bytes,1,opt,name=resource_name,json=resourceName,proto3" json:"resource_name,omitempty"`
	// The Keyword Plan ad group to which this keyword belongs.
	KeywordPlanAdGroup *wrappers.StringValue `protobuf:"bytes,2,opt,name=keyword_plan_ad_group,json=keywordPlanAdGroup,proto3" json:"keyword_plan_ad_group,omitempty"`
	// The ID of the Keyword Plan keyword.
	Id *wrappers.Int64Value `protobuf:"bytes,3,opt,name=id,proto3" json:"id,omitempty"`
	// The keyword text.
	Text *wrappers.StringValue `protobuf:"bytes,4,opt,name=text,proto3" json:"text,omitempty"`
	// The keyword match type.
	MatchType enums.KeywordMatchTypeEnum_KeywordMatchType `protobuf:"varint,5,opt,name=match_type,json=matchType,proto3,enum=google.ads.googleads.v0.enums.KeywordMatchTypeEnum_KeywordMatchType" json:"match_type,omitempty"`
	// A keyword level max cpc bid in micros, in the account currency, that
	// overrides the keyword plan ad group cpc bid.
	CpcBidMicros         *wrappers.Int64Value `protobuf:"bytes,6,opt,name=cpc_bid_micros,json=cpcBidMicros,proto3" json:"cpc_bid_micros,omitempty"`
	XXX_NoUnkeyedLiteral struct{}             `json:"-"`
	XXX_unrecognized     []byte               `json:"-"`
	XXX_sizecache        int32                `json:"-"`
}

func (m *KeywordPlanKeyword) Reset()         { *m = KeywordPlanKeyword{} }
func (m *KeywordPlanKeyword) String() string { return proto.CompactTextString(m) }
func (*KeywordPlanKeyword) ProtoMessage()    {}
func (*KeywordPlanKeyword) Descriptor() ([]byte, []int) {
	return fileDescriptor_keyword_plan_keyword_f025d0423a196b23, []int{0}
}
func (m *KeywordPlanKeyword) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_KeywordPlanKeyword.Unmarshal(m, b)
}
func (m *KeywordPlanKeyword) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_KeywordPlanKeyword.Marshal(b, m, deterministic)
}
func (dst *KeywordPlanKeyword) XXX_Merge(src proto.Message) {
	xxx_messageInfo_KeywordPlanKeyword.Merge(dst, src)
}
func (m *KeywordPlanKeyword) XXX_Size() int {
	return xxx_messageInfo_KeywordPlanKeyword.Size(m)
}
func (m *KeywordPlanKeyword) XXX_DiscardUnknown() {
	xxx_messageInfo_KeywordPlanKeyword.DiscardUnknown(m)
}

var xxx_messageInfo_KeywordPlanKeyword proto.InternalMessageInfo

func (m *KeywordPlanKeyword) GetResourceName() string {
	if m != nil {
		return m.ResourceName
	}
	return ""
}

func (m *KeywordPlanKeyword) GetKeywordPlanAdGroup() *wrappers.StringValue {
	if m != nil {
		return m.KeywordPlanAdGroup
	}
	return nil
}

func (m *KeywordPlanKeyword) GetId() *wrappers.Int64Value {
	if m != nil {
		return m.Id
	}
	return nil
}

func (m *KeywordPlanKeyword) GetText() *wrappers.StringValue {
	if m != nil {
		return m.Text
	}
	return nil
}

func (m *KeywordPlanKeyword) GetMatchType() enums.KeywordMatchTypeEnum_KeywordMatchType {
	if m != nil {
		return m.MatchType
	}
	return enums.KeywordMatchTypeEnum_UNSPECIFIED
}

func (m *KeywordPlanKeyword) GetCpcBidMicros() *wrappers.Int64Value {
	if m != nil {
		return m.CpcBidMicros
	}
	return nil
}

func init() {
	proto.RegisterType((*KeywordPlanKeyword)(nil), "google.ads.googleads.v0.resources.KeywordPlanKeyword")
}

func init() {
	proto.RegisterFile("google/ads/googleads/v0/resources/keyword_plan_keyword.proto", fileDescriptor_keyword_plan_keyword_f025d0423a196b23)
}

var fileDescriptor_keyword_plan_keyword_f025d0423a196b23 = []byte{
	// 439 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0x8c, 0x92, 0xdf, 0x8a, 0xd4, 0x30,
	0x14, 0xc6, 0x69, 0x67, 0x5d, 0xd8, 0xb8, 0xee, 0x45, 0x40, 0x2c, 0xab, 0xc8, 0xac, 0xb2, 0x30,
	0x20, 0xa4, 0x65, 0x95, 0xbd, 0x88, 0xde, 0x74, 0x50, 0x06, 0x95, 0xd5, 0x61, 0x94, 0xb9, 0x90,
	0x42, 0xc9, 0x24, 0x31, 0x96, 0x6d, 0xfe, 0x90, 0xb4, 0xbb, 0xce, 0xbd, 0x2f, 0xe0, 0x2b, 0x78,
	0xe9, 0xa3, 0xf8, 0x28, 0x3e, 0x85, 0xb4, 0x69, 0x3b, 0xc8, 0xb0, 0x8e, 0x77, 0x5f, 0x92, 0xef,
	0x77, 0xbe, 0x9e, 0xd3, 0x03, 0x5e, 0x08, 0xad, 0x45, 0xc9, 0x63, 0xc2, 0x5c, 0xec, 0x65, 0xa3,
	0xae, 0x92, 0xd8, 0x72, 0xa7, 0x6b, 0x4b, 0xb9, 0x8b, 0x2f, 0xf9, 0xfa, 0x5a, 0x5b, 0x96, 0x9b,
	0x92, 0xa8, 0xbc, 0x3b, 0x20, 0x63, 0x75, 0xa5, 0xe1, 0x89, 0x47, 0x10, 0x61, 0x0e, 0x0d, 0x34,
	0xba, 0x4a, 0xd0, 0x40, 0x1f, 0x9f, 0xdf, 0x14, 0xc0, 0x55, 0x2d, 0x37, 0xc5, 0x25, 0xa9, 0xe8,
	0x97, 0xbc, 0x5a, 0x1b, 0xee, 0x4b, 0x1f, 0x3f, 0xec, 0xb8, 0xf6, 0xb4, 0xaa, 0x3f, 0xc7, 0xd7,
	0x96, 0x18, 0xc3, 0xad, 0xf3, 0xef, 0x8f, 0xbe, 0x8f, 0x00, 0x7c, 0xeb, 0xe1, 0x79, 0x49, 0x54,
	0x27, 0xe1, 0x63, 0x70, 0xa7, 0xcf, 0xce, 0x15, 0x91, 0x3c, 0x0a, 0xc6, 0xc1, 0xe4, 0x60, 0x71,
	0xd8, 0x5f, 0xbe, 0x23, 0x92, 0xc3, 0xf7, 0xe0, 0xee, 0x5f, 0x4d, 0x11, 0x96, 0x0b, 0xab, 0x6b,
	0x13, 0x85, 0xe3, 0x60, 0x72, 0xfb, 0xec, 0x41, 0xd7, 0x0b, 0xea, 0xb3, 0xd1, 0x87, 0xca, 0x16,
	0x4a, 0x2c, 0x49, 0x59, 0xf3, 0x05, 0xbc, 0xdc, 0xa4, 0xa6, 0x6c, 0xd6, 0x70, 0xf0, 0x09, 0x08,
	0x0b, 0x16, 0x8d, 0x5a, 0xfa, 0xfe, 0x16, 0xfd, 0x5a, 0x55, 0xe7, 0xcf, 0x3c, 0x1c, 0x16, 0x0c,
	0x26, 0x60, 0xaf, 0xe2, 0x5f, 0xab, 0x68, 0xef, 0x3f, 0xc2, 0x5a, 0x27, 0xa4, 0x00, 0x6c, 0xe6,
	0x13, 0xdd, 0x1a, 0x07, 0x93, 0xa3, 0xb3, 0x97, 0xe8, 0xa6, 0xd9, 0xb7, 0x83, 0x45, 0xdd, 0x40,
	0x2e, 0x1a, 0xee, 0xe3, 0xda, 0xf0, 0x57, 0xaa, 0x96, 0x5b, 0x97, 0x8b, 0x03, 0xd9, 0x4b, 0x98,
	0x82, 0x23, 0x6a, 0x68, 0xbe, 0x2a, 0x58, 0x2e, 0x0b, 0x6a, 0xb5, 0x8b, 0xf6, 0x77, 0xf7, 0x73,
	0x48, 0x0d, 0x9d, 0x16, 0xec, 0xa2, 0x05, 0xa6, 0xdf, 0x42, 0x70, 0x4a, 0xb5, 0x44, 0x3b, 0xb7,
	0x62, 0x7a, 0x6f, 0xfb, 0xd7, 0xcd, 0x9b, 0xf2, 0xf3, 0xe0, 0xd3, 0x9b, 0x8e, 0x16, 0xba, 0x24,
	0x4a, 0x20, 0x6d, 0x45, 0x2c, 0xb8, 0x6a, 0xc3, 0xfb, 0x05, 0x32, 0x85, 0xfb, 0xc7, 0xc2, 0x3e,
	0x1f, 0xd4, 0x8f, 0x70, 0x34, 0x4b, 0xd3, 0x9f, 0xe1, 0xc9, 0xcc, 0x97, 0x4c, 0x99, 0x43, 0x5e,
	0x36, 0x6a, 0x99, 0xa0, 0x45, 0xef, 0xfc, 0xd5, 0x7b, 0xb2, 0x94, 0xb9, 0x6c, 0xf0, 0x64, 0xcb,
	0x24, 0x1b, 0x3c, 0xbf, 0xc3, 0x53, 0xff, 0x80, 0x71, 0xca, 0x1c, 0xc6, 0x83, 0x0b, 0xe3, 0x65,
	0x82, 0xf1, 0xe0, 0x5b, 0xed, 0xb7, 0x1f, 0xfb, 0xf4, 0x4f, 0x00, 0x00, 0x00, 0xff, 0xff, 0x13,
	0xc4, 0xff, 0xa5, 0x5c, 0x03, 0x00, 0x00,
}
