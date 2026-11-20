package jsoncodec

import gtjson "github.com/gtkit/json/v2"

// Codec adapts gtkit/json/v2 to httpc.JSONCodec.
type Codec struct{}

// Marshal encodes a value with gtkit/json/v2.
func (Codec) Marshal(v any) ([]byte, error) { return gtjson.Marshal(v) }

// Unmarshal decodes bytes with gtkit/json/v2.
func (Codec) Unmarshal(data []byte, v any) error { return gtjson.Unmarshal(data, v) }
