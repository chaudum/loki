package ingester

import (
	"bytes"
	"context"
	"io"
	"time"

	"github.com/apache/arrow-go/v18/arrow/array"
	"github.com/apache/arrow-go/v18/arrow/memory"
	"github.com/apache/arrow-go/v18/parquet/file"
	"github.com/apache/arrow-go/v18/parquet/pqarrow"
	"github.com/grafana/loki/v3/pkg/logproto"
	"github.com/parquet-go/parquet-go"
)

type ParquetRow struct {
	Timestamp time.Time         `parquet:"timestamp,timestamp(millisecond),delta"`
	Message   string            `parquet:"message,zstd"`
	Metadata  map[string]string `parquet:"metadata"`
}

func NewParquetChunk(size int) *ParquetChunk {
	buf := bytes.NewBuffer(make([]byte, 0, size))
	return &ParquetChunk{
		data: buf,
		w:    parquet.NewGenericWriter[ParquetRow](buf),
	}
}

type ParquetChunk struct {
	data *bytes.Buffer
	w    *parquet.GenericWriter[ParquetRow]
}

func (pc *ParquetChunk) hasSpaceFor(entry *logproto.Entry) bool {
	const maxSize = 4 << 20 // 4MiB
	return pc.data.Len() < maxSize
}

func (pc *ParquetChunk) Append(entry *logproto.Entry) error {
	metadata := make(map[string]string, len(entry.StructuredMetadata))
	for _, lbl := range entry.StructuredMetadata {
		metadata[lbl.Name] = lbl.Value
	}
	row := ParquetRow{
		Timestamp: entry.Timestamp,
		Message:   entry.Line,
		Metadata:  metadata,
	}
	_, err := pc.w.Write([]ParquetRow{row})
	return err
}

func (pc *ParquetChunk) Close() error {
	return pc.w.Close()
}

func (pc *ParquetChunk) Reset() {
	pc.data.Reset()
}

func (pc *ParquetChunk) Reader() io.Reader {
	return pc.data
}

func (pc *ParquetChunk) RecordReader(ctx context.Context) (array.RecordReader, error) {
	return recordReaderFromBytes(ctx, pc.data.Bytes())
}

func recordReaderFromBytes(ctx context.Context, data []byte) (array.RecordReader, error) {
	// Create a ReaderAtSeeker from byte slice
	reader := bytes.NewReader(data)

	// Open the parquet file
	parquetReader, err := file.NewParquetReader(reader)
	if err != nil {
		return nil, err
	}

	// Create Arrow FileReader
	arrowReader, err := pqarrow.NewFileReader(
		parquetReader,
		pqarrow.ArrowReadProperties{
			Parallel:  true, // Read columns in parallel
			BatchSize: 1024, // Batch size for reading
		},
		memory.DefaultAllocator,
	)
	if err != nil {
		return nil, err
	}

	// Get RecordReader for all columns and all row groups
	recordReader, err := arrowReader.GetRecordReader(ctx, nil, nil)
	if err != nil {
		return nil, err
	}

	return recordReader, nil
}
