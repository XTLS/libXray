package xray

import (
	"context"
	"fmt"
	"reflect"

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

func writeResult(m proto.Message) (string, error) {
	if isNil(m) {
		return "", fmt.Errorf("m is nil")
	}
	ops := protojson.MarshalOptions{}
	b, err := ops.Marshal(m)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// query system stats and outbound stats.
// server means The API server address, like "127.0.0.1:8080".
// dir means the dir which result json will be wrote to.
func QueryStats(server string) (string, string, error) {
	conn, err := grpc.NewClient(server, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return "", "", err
	}
	defer conn.Close()

	client := statsService.NewStatsServiceClient(conn)

	sysStatsReq := &statsService.SysStatsRequest{}
	sysStatsRes, err := client.GetSysStats(context.Background(), sysStatsReq)
	if err != nil {
		return "", "", err
	}
	sysStatsJson, err := writeResult(sysStatsRes)
	if err != nil {
		return "", "", err
	}

	statsReq := &statsService.QueryStatsRequest{
		Pattern: "",
		Reset_:  false,
	}
	statsRes, err := client.QueryStats(context.Background(), statsReq)
	if err != nil {
		return "", "", err
	}
	statsJson, err := writeResult(statsRes)
	if err != nil {
		return "", "", err
	}
	return sysStatsJson, statsJson, nil
}
