package pdf

import (
	"context"
)

type Service interface {
	GenerateDocx(ctx context.Context, req *DocxRequest) ([]byte, error)
}
