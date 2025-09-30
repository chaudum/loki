package iceberg

import (
	"context"

	"github.com/apache/iceberg-go"
	"github.com/apache/iceberg-go/catalog"
	"github.com/apache/iceberg-go/table"
	"github.com/grafana/loki/v3/pkg/logproto"
	"github.com/grafana/loki/v3/pkg/storage/chunk"
	"github.com/grafana/loki/v3/pkg/storage/stores/index"
	"github.com/grafana/loki/v3/pkg/storage/stores/index/stats"
	"github.com/grafana/loki/v3/pkg/storage/stores/shipper/indexshipper/tsdb/sharding"
	"github.com/prometheus/common/model"
	"github.com/prometheus/prometheus/model/labels"
)

type Index struct {
	endpoint string
}

var _ index.ReaderWriter = &Index{}

func NewIndex() *Index {
	return &Index{
		endpoint: "http://localhost:8181",
	}
}

// IndexChunk implements index.ReaderWriter.
func (i *Index) IndexChunk(ctx context.Context, from model.Time, through model.Time, chk chunk.Chunk) error {
	panic("unimplemented")
}

// GetSeries implements index.Reader.
func (i *Index) GetSeries(ctx context.Context, userID string, from model.Time, through model.Time, matchers ...*labels.Matcher) ([]labels.Labels, error) {
	panic("unimplemented")
}

// GetShards implements index.Reader.
func (i *Index) GetShards(ctx context.Context, userID string, from model.Time, through model.Time, targetBytesPerShard uint64, predicate chunk.Predicate) (*logproto.ShardsResponse, error) {
	panic("unimplemented")
}

// HasForSeries implements index.Reader.
func (i *Index) HasForSeries(from model.Time, through model.Time) (sharding.ForSeries, bool) {
	panic("unimplemented")
}

// LabelNamesForMetricName implements index.Reader.
func (i *Index) LabelNamesForMetricName(ctx context.Context, userID string, from model.Time, through model.Time, metricName string, matchers ...*labels.Matcher) ([]string, error) {
	panic("unimplemented")
}

// LabelValuesForMetricName implements index.Reader.
func (i *Index) LabelValuesForMetricName(ctx context.Context, userID string, from model.Time, through model.Time, metricName string, labelName string, matchers ...*labels.Matcher) ([]string, error) {
	panic("unimplemented")
}

// SetChunkFilterer implements index.Reader.
func (i *Index) SetChunkFilterer(chunkFilter chunk.RequestChunkFilterer) {
	panic("unimplemented")
}

// Stats implements index.Reader.
func (i *Index) Stats(ctx context.Context, userID string, from model.Time, through model.Time, matchers ...*labels.Matcher) (*stats.Stats, error) {
	panic("unimplemented")
}

// Volume implements index.Reader.
func (i *Index) Volume(ctx context.Context, userID string, from model.Time, through model.Time, limit int32, targetLabels []string, aggregateBy string, matchers ...*labels.Matcher) (*logproto.VolumeResponse, error) {
	panic("unimplemented")
}

func (i *Index) GetChunkRefs(ctx context.Context, userID string, from, through model.Time, predicate chunk.Predicate) ([]logproto.ChunkRef, error) {
	// Load catalog and table
	cat, err := catalog.Load(ctx, "logslake", iceberg.Properties{"uri": i.endpoint})
	if err != nil {
		return nil, err
	}

	tbl, err := cat.LoadTable(ctx, table.Identifier{"loki", "chunks"})
	if err != nil {
		return nil, err
	}

	opts := make([]table.ScanOption, 0)
	// TODO: select chunks by predicate

	scan := tbl.Scan(opts...)

	tasks, err := scan.PlanFiles(ctx)
	if err != nil {
		return nil, err
	}

	chunkRefs := make([]logproto.ChunkRef, 0, len(tasks))
	for _, task := range tasks {
		c, err := chunk.ParseExternalKey(userID, task.File.FilePath())
		if err != nil {
			return nil, err
		}
		chunkRefs = append(chunkRefs, c.ChunkRef)
	}
	return chunkRefs, nil
}
