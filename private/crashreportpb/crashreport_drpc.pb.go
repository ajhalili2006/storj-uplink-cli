// Code generated by protoc-gen-go-drpc. DO NOT EDIT.
// protoc-gen-go-drpc version: v0.0.34
// source: crashreport.proto

package crashreportpb

import (
	bytes "bytes"
	context "context"
	errors "errors"

	jsonpb "github.com/gogo/protobuf/jsonpb"
	proto "github.com/gogo/protobuf/proto"

	drpc "storj.io/drpc"
	drpcerr "storj.io/drpc/drpcerr"
)

type drpcEncoding_File_crashreport_proto struct{}

func (drpcEncoding_File_crashreport_proto) Marshal(msg drpc.Message) ([]byte, error) {
	return proto.Marshal(msg.(proto.Message))
}

func (drpcEncoding_File_crashreport_proto) Unmarshal(buf []byte, msg drpc.Message) error {
	return proto.Unmarshal(buf, msg.(proto.Message))
}

func (drpcEncoding_File_crashreport_proto) JSONMarshal(msg drpc.Message) ([]byte, error) {
	var buf bytes.Buffer
	err := new(jsonpb.Marshaler).Marshal(&buf, msg.(proto.Message))
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (drpcEncoding_File_crashreport_proto) JSONUnmarshal(buf []byte, msg drpc.Message) error {
	return jsonpb.Unmarshal(bytes.NewReader(buf), msg.(proto.Message))
}

type DRPCCrashReportClient interface {
	DRPCConn() drpc.Conn

	Report(ctx context.Context, in *ReportRequest) (*ReportResponse, error)
}

type drpcCrashReportClient struct {
	cc drpc.Conn
}

func NewDRPCCrashReportClient(cc drpc.Conn) DRPCCrashReportClient {
	return &drpcCrashReportClient{cc}
}

func (c *drpcCrashReportClient) DRPCConn() drpc.Conn { return c.cc }

func (c *drpcCrashReportClient) Report(ctx context.Context, in *ReportRequest) (*ReportResponse, error) {
	out := new(ReportResponse)
	err := c.cc.Invoke(ctx, "/crash.CrashReport/Report", drpcEncoding_File_crashreport_proto{}, in, out)
	if err != nil {
		return nil, err
	}
	return out, nil
}

type DRPCCrashReportServer interface {
	Report(context.Context, *ReportRequest) (*ReportResponse, error)
}

type DRPCCrashReportUnimplementedServer struct{}

func (s *DRPCCrashReportUnimplementedServer) Report(context.Context, *ReportRequest) (*ReportResponse, error) {
	return nil, drpcerr.WithCode(errors.New("Unimplemented"), drpcerr.Unimplemented)
}

type DRPCCrashReportDescription struct{}

func (DRPCCrashReportDescription) NumMethods() int { return 1 }

func (DRPCCrashReportDescription) Method(n int) (string, drpc.Encoding, drpc.Receiver, interface{}, bool) {
	switch n {
	case 0:
		return "/crash.CrashReport/Report", drpcEncoding_File_crashreport_proto{},
			func(srv interface{}, ctx context.Context, in1, in2 interface{}) (drpc.Message, error) {
				return srv.(DRPCCrashReportServer).
					Report(
						ctx,
						in1.(*ReportRequest),
					)
			}, DRPCCrashReportServer.Report, true
	default:
		return "", nil, nil, nil, false
	}
}

func DRPCRegisterCrashReport(mux drpc.Mux, impl DRPCCrashReportServer) error {
	return mux.Register(impl, DRPCCrashReportDescription{})
}

type DRPCCrashReport_ReportStream interface {
	drpc.Stream
	SendAndClose(*ReportResponse) error
}

type drpcCrashReport_ReportStream struct {
	drpc.Stream
}

func (x *drpcCrashReport_ReportStream) SendAndClose(m *ReportResponse) error {
	if err := x.MsgSend(m, drpcEncoding_File_crashreport_proto{}); err != nil {
		return err
	}
	return x.CloseSend()
}
