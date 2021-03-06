package confita_test

import (
	"context"
	"encoding/json"
	"math"
	"strconv"
	"testing"
	"time"

	"github.com/heetch/confita"
	"github.com/heetch/confita/backend"
	"github.com/stretchr/testify/require"
)

type store map[string]string

func (s store) Get(ctx context.Context, key string) ([]byte, error) {
	data, ok := s[key]
	if !ok {
		return nil, backend.ErrNotFound
	}

	return []byte(data), nil
}

type longRunningStore time.Duration

func (s longRunningStore) Get(ctx context.Context, key string) ([]byte, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-time.After(time.Duration(s)):
		return []byte(time.Now().String()), nil
	}
}

type valueUnmarshaler store

func (k valueUnmarshaler) Get(ctx context.Context, key string) ([]byte, error) {
	return store(k).Get(ctx, key)
}

func (k valueUnmarshaler) UnmarshalValue(ctx context.Context, key string, to interface{}) error {
	data, err := store(k).Get(ctx, key)
	if err != nil {
		return err
	}

	return json.Unmarshal(data, to)
}

func TestLoad(t *testing.T) {
	type nested struct {
		Int    int    `config:"int"`
		String string `config:"string"`
	}

	type testStruct struct {
		Bool            bool          `config:"bool"`
		Int             int           `config:"int"`
		Int8            int8          `config:"int8"`
		Int16           int16         `config:"int16"`
		Int32           int32         `config:"int32"`
		Int64           int64         `config:"int64"`
		Uint            uint          `config:"uint"`
		Uint8           uint8         `config:"uint8"`
		Uint16          uint16        `config:"uint16"`
		Uint32          uint32        `config:"uint32"`
		Uint64          uint64        `config:"uint64"`
		Float32         float32       `config:"float32"`
		Float64         float64       `config:"float64"`
		Ptr             *string       `config:"ptr"`
		String          string        `config:"string"`
		Duration        time.Duration `config:"duration"`
		Struct          nested
		StructPtrNil    *nested
		StructPtrNotNil *nested
		Ignored         string
	}

	var s testStruct
	s.StructPtrNotNil = new(nested)

	boolStore := store{
		"bool": "true",
	}

	intStore := store{
		"int":   strconv.FormatInt(math.MaxInt64, 10),
		"int8":  strconv.FormatInt(math.MaxInt8, 10),
		"int16": strconv.FormatInt(math.MaxInt16, 10),
		"int32": strconv.FormatInt(math.MaxInt32, 10),
		"int64": strconv.FormatInt(math.MaxInt64, 10),
	}

	uintStore := store{
		"uint":   strconv.FormatUint(math.MaxUint64, 10),
		"uint8":  strconv.FormatUint(math.MaxUint8, 10),
		"uint16": strconv.FormatUint(math.MaxUint16, 10),
		"uint32": strconv.FormatUint(math.MaxUint32, 10),
		"uint64": strconv.FormatUint(math.MaxUint64, 10),
	}

	floatStore := store{
		"float32": strconv.FormatFloat(math.MaxFloat32, 'f', 6, 32),
		"float64": strconv.FormatFloat(math.MaxFloat64, 'f', 6, 64),
	}

	otherStore := store{
		"ptr":      "ptr",
		"string":   "string",
		"duration": "10s",
	}

	loader := confita.NewLoader(
		boolStore,
		intStore,
		uintStore,
		floatStore,
		otherStore)
	err := loader.Load(context.Background(), &s)
	require.NoError(t, err)

	ptr := "ptr"
	require.EqualValues(t, testStruct{
		Bool:     true,
		Int:      math.MaxInt64,
		Int8:     math.MaxInt8,
		Int16:    math.MaxInt16,
		Int32:    math.MaxInt32,
		Int64:    math.MaxInt64,
		Uint:     math.MaxUint64,
		Uint8:    math.MaxUint8,
		Uint16:   math.MaxUint16,
		Uint32:   math.MaxUint32,
		Uint64:   math.MaxUint64,
		Float32:  math.MaxFloat32,
		Float64:  math.MaxFloat64,
		Ptr:      &ptr,
		String:   "string",
		Duration: 10 * time.Second,
		Struct: nested{
			Int:    math.MaxInt64,
			String: "string",
		},
		StructPtrNotNil: &nested{
			Int:    math.MaxInt64,
			String: "string",
		},
	}, s)
}

func TestLoadRequired(t *testing.T) {
	s := struct {
		Name string `config:"name,required"`
	}{}

	st := make(store)
	err := confita.NewLoader(st).Load(context.Background(), &s)
	require.Error(t, err)
}

func TestLoadIgnored(t *testing.T) {
	s := struct {
		Name string `config:"-"`
		Age  int    `config:"age"`
	}{}

	st := store{
		"name": "name",
		"age":  "10",
	}

	err := confita.NewLoader(st).Load(context.Background(), &s)
	require.NoError(t, err)
	require.Equal(t, 10, s.Age)
	require.Zero(t, s.Name)
}

func TestLoadContextCancel(t *testing.T) {
	s := struct {
		Name string `config:"-"`
		Age  int    `config:"age"`
	}{}

	st := store{
		"name": "name",
		"age":  "10",
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err := confita.NewLoader(st).Load(ctx, &s)
	require.Equal(t, context.Canceled, err)
}

func TestLoadContextTimeout(t *testing.T) {
	s := struct {
		Name string `config:"-"`
		Age  int    `config:"age"`
	}{}

	st := longRunningStore(10 * time.Second)
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	err := confita.NewLoader(st).Load(ctx, &s)
	require.Equal(t, context.DeadlineExceeded, err)
}

func TestLoadFromValueUnmarshaler(t *testing.T) {
	s := struct {
		Name    string `config:"name"`
		Age     int    `config:"age"`
		Ignored string `config:"-"`
	}{}

	st := valueUnmarshaler{
		"name": `"name"`,
		"age":  "10",
	}

	err := confita.NewLoader(st).Load(context.Background(), &s)
	require.NoError(t, err)
	require.Equal(t, "name", s.Name)
	require.Equal(t, 10, s.Age)
	require.Zero(t, s.Ignored)
}
