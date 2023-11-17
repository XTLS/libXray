package xray

import (
	"context"
	"fmt"
	"path"
	"reflect"

	"github.com/xtls/libxray/nodep"
	statsService "github.com/xtls/xray-core/app/stats/command"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

func isNil(i interface{}) bool {
	vi := reflect.ValueOf(i)
	if vi.Kind() == reflect.Ptr {
		return vi.IsNil()
	}
	return i == nil
}

func writeResult(m proto.Message, path string) error {
	if isNil(m) {
		return fmt.Errorf("m is nil")
	}
	ops := protojson.MarshalOptions{}
	b, err := ops.Marshal(m)
	if err != nil {
		return err
	}
	err = nodep.WriteBytes(b, path)
	return err
}

// query system stats and outbound stats.
// server means The API server address, like "127.0.0.1:8080".
// dir means the dir which result json will be wrote to.
func QueryStats(server string, dir string) string {
	conn, err := grpc.Dial(server, grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithBlock())
	if err != nil {
		return err.Error()
	}
	defer conn.Close()

	client := statsService.NewStatsServiceClient(conn)

	sysStatsReq := &statsService.SysStatsRequest{}
	sysStatsRes, err := client.GetSysStats(context.Background(), sysStatsReq)
	if err != nil {
		return err.Error()
	}
	sysStatsPath := path.Join(dir, "sysStats.json")
	err = writeResult(sysStatsRes, sysStatsPath)
	if err != nil {
		return err.Error()
	}

	statsReq := &statsService.QueryStatsRequest{
		Pattern: "",
		Reset_:  false,
	}
	statsRes, err := client.QueryStats(context.Background(), statsReq)
	if err != nil {
		return err.Error()
	}
	statsPath := path.Join(dir, "stats.json")
	err = writeResult(statsRes, statsPath)
	if err != nil {
		return err.Error()
	}
	return ""
}
