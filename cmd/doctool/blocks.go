package main

import (
	"flag"
	"reflect"

	"github.com/grafana/dskit/kv/consul"
	"github.com/grafana/dskit/kv/etcd"
	"github.com/grafana/dskit/kv/memberlist"
	"github.com/grafana/dskit/runtimeconfig"
	"github.com/weaveworks/common/server"

	"github.com/grafana/loki/pkg/distributor"
	"github.com/grafana/loki/pkg/ingester"
	ingesterclient "github.com/grafana/loki/pkg/ingester/client"
	"github.com/grafana/loki/pkg/loki"
	"github.com/grafana/loki/pkg/loki/common"
	"github.com/grafana/loki/pkg/lokifrontend"
	"github.com/grafana/loki/pkg/querier"
	"github.com/grafana/loki/pkg/querier/queryrange"
	"github.com/grafana/loki/pkg/querier/worker"
	"github.com/grafana/loki/pkg/ruler"
	"github.com/grafana/loki/pkg/scheduler"
	"github.com/grafana/loki/pkg/storage"
	"github.com/grafana/loki/pkg/storage/chunk"
	"github.com/grafana/loki/pkg/storage/chunk/client/aws"
	"github.com/grafana/loki/pkg/storage/chunk/client/azure"
	"github.com/grafana/loki/pkg/storage/chunk/client/gcp"
	"github.com/grafana/loki/pkg/storage/chunk/client/hedging"
	"github.com/grafana/loki/pkg/storage/chunk/client/openstack"
	"github.com/grafana/loki/pkg/storage/config"
	"github.com/grafana/loki/pkg/storage/stores/series/index"
	"github.com/grafana/loki/pkg/storage/stores/shipper/compactor"
	"github.com/grafana/loki/pkg/tracing"
	"github.com/grafana/loki/pkg/usagestats"
	"github.com/grafana/loki/pkg/validation"
)

type Block struct {
	Name string
	Desc string
	Type reflect.Type
}

type Configuration interface {
	RegisterFlags(*flag.FlagSet)
}

func Config() Configuration {
	return &loki.Config{}
}

func Blocks() []Block {
	return []Block{
		{
			Name: "common_config",
			Type: reflect.TypeOf(common.Config{}),
			Desc: "Common configuration shared between multiple modules.\nIf a more specific configuration is given in other sections, the related configuration within thie section will be ignored.",
		},
		{
			Name: "server_config",
			Type: reflect.TypeOf(server.Config{}),
			Desc: "Configures the HTTP/gRPC server of the started module(s).",
		},
		{
			Name: "distributor_config",
			Type: reflect.TypeOf(distributor.Config{}),
			Desc: "Configures the distributors and how they connect to each other via the ring.",
		},
		{
			Name: "querier_config",
			Type: reflect.TypeOf(querier.Config{}),
			Desc: "Configures the queriers\nOnly applicable when running target `all` or `querier`.",
		},
		{
			Name: "ingester_config",
			Type: reflect.TypeOf(ingester.Config{}),
			Desc: "Configures the ingesters and how they register themselves to the distributor ring.",
		},
		{
			Name: "ingester_client_config",
			Type: reflect.TypeOf(ingesterclient.Config{}),
			Desc: "Configures how the ingester clients on the distributors and queriers will connect to the ingesters\nOnly applicable when running target `all`, `distributor`, or `querier`.",
		},
		{
			Name: "storage_config",
			Type: reflect.TypeOf(storage.Config{}),
			Desc: "Configures where Loki stores data.",
		},
		{
			Name: "chunkstore_config",
			Type: reflect.TypeOf(chunk.Config{}),
			Desc: "Configures how Loki stores data in the specific store.",
		},
		{
			Name: "schema_config",
			Type: reflect.TypeOf(config.SchemaConfig{}),
			Desc: "Configures the chunk index schema and where it is stored.",
		},
		{
			Name: "limits_config",
			Type: reflect.TypeOf(validation.Limits{}),
			Desc: "Configures per-tenant or global limits.",
		},
		{
			Name: "table_manager_config",
			Type: reflect.TypeOf(index.TableManagerConfig{}),
			Desc: "Configures how long the table manager retains data.",
		},
		{
			Name: "frontend_worker_config",
			Type: reflect.TypeOf(worker.Config{}),
			Desc: "Configures the workers that are running with in the queriers and picking up and executing queries enqued by the query frontend.",
		},
		{
			Name: "frontend_config",
			Type: reflect.TypeOf(lokifrontend.Config{}),
			Desc: "Configures the query frontends.",
		},
		{
			Name: "ruler_config",
			Type: reflect.TypeOf(ruler.Config{}),
			Desc: "Configures the rulers.",
		},
		{
			Name: "query_range_config",
			Type: reflect.TypeOf(queryrange.Config{}),
			Desc: "Configures how the query frontend splits and caches queries.",
		},
		{
			Name: "runtime_config",
			Type: reflect.TypeOf(runtimeconfig.Config{}),
			Desc: "Configures the runtime config file and how often it is reloaded.",
		},
		{
			Name: "memberlist_config",
			Type: reflect.TypeOf(memberlist.KVConfig{}),
			Desc: "Configures how ring members communicate with each other using memberlist.",
		},
		{
			Name: "tracing_config",
			Type: reflect.TypeOf(tracing.Config{}),
			Desc: "Configures tracing options.",
		},
		{
			Name: "compactor_config",
			Type: reflect.TypeOf(compactor.Config{}),
			Desc: "Configures how the compactors compact index shards for performance.",
		},
		{
			Name: "query_scheduler_config",
			Type: reflect.TypeOf(scheduler.Config{}),
			Desc: "Configures the query schedulers.\nWhen configured, tenant query queues are separated from the query frontend.",
		},
		{
			Name: "analytics_config",
			Type: reflect.TypeOf(usagestats.Config{}),
			Desc: "Configures how anonymous usage data is sent to grafana.com.",
		},

		// common configuration blocks

		{
			Name: "consul_config",
			Type: reflect.TypeOf(consul.Config{}),
			Desc: "Configures the Consul client.",
		},
		{
			Name: "etcd_config",
			Type: reflect.TypeOf(etcd.Config{}),
			Desc: "Configures the etcd client.",
		},

		// common storage blocks

		{
			Name: "azure_storage_config",
			Type: reflect.TypeOf(azure.BlobStorageConfig{}),
			Desc: "Configures the client for Azure Blob Storage as storage.",
		},
		{
			Name: "gcs_storage_config",
			Type: reflect.TypeOf(gcp.GCSConfig{}),
			Desc: "Configures the client for GCS as storage.",
		},
		{
			Name: "s3_storage_config",
			Type: reflect.TypeOf(aws.S3Config{}),
			Desc: "Configures the client Amazon S3 as storage",
		},
		{
			Name: "swift_storage_config",
			Type: reflect.TypeOf(openstack.SwiftConfig{}),
			Desc: "Configures Swift as storage",
		},
		{
			Name: "filesystem_storage_config",
			Type: reflect.TypeOf(common.FilesystemConfig{}),
			Desc: "Configures a (local) file system as storage",
		},
		{
			Name: "hedging_config",
			Type: reflect.TypeOf(hedging.Config{}),
			Desc: "Configures how to hedge requests for the storage",
		},
	}
}
