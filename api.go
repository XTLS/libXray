package libXray

import (
	"context"
	"fmt"
	"path"
	"reflect"
	"time"

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
// server means The API server address.
// dir means the dir which result json will be wrote to.
func QueryStats(server string, dir string) string {
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(3)*time.Second)
	conn, err := grpc.DialContext(ctx, server, grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithBlock())
	close := func() {
		cancel()
		conn.Close()
	}
	defer close()
	if err != nil {
		return err.Error()
	}

	client := statsService.NewStatsServiceClient(conn)

	sysStatsReq := &statsService.SysStatsRequest{}
	sysStatsRes, err := client.GetSysStats(ctx, sysStatsReq)
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
	statsRes, err := client.QueryStats(ctx, statsReq)
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
