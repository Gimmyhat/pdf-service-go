package pdf

import "context"

type Service interface {
	// ... existing code ...
	GenerateDocx(ctx context.Context, req *DocxRequest) ([]byte, error)
}
