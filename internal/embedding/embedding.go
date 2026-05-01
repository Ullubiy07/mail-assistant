package embedding

type Chunk = string
type Embedding = []float32

type Client interface {
	Get(chunks []Chunk) ([]Embedding, error)
}
