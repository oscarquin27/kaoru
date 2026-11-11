package usecase

import (
	"context"

	shared "kaoru/internal/shared/domain"
)

const verifyBatch = 1000

// VerifyChain recalcula la cadena de hash y dice si alguien alteró un registro.
type VerifyChain struct {
	d Deps
}

// VerifyChainInput es la entrada de VerifyChain.
type VerifyChainInput struct {
	Actor shared.Actor
}

// VerifyChainOutput es el resultado de la verificación.
type VerifyChainOutput struct {
	Intact  bool
	Entries int
	// FirstBrokenSeq es el primer registro que no cuadra; 0 si la cadena está íntegra.
	FirstBrokenSeq int64
}

// Execute exige el permiso audit.read y recorre la cadena por lotes.
func (uc *VerifyChain) Execute(ctx context.Context, in VerifyChainInput) (VerifyChainOutput, error) {
	if err := uc.d.Permissions.Require(ctx, in.Actor.Roles, PermissionRead); err != nil {
		return VerifyChainOutput{}, err
	}
	out := VerifyChainOutput{Intact: true}
	var afterSeq int64
	prev := ""
	for {
		batch, err := uc.d.Repository.ChainAfter(ctx, afterSeq, verifyBatch)
		if err != nil {
			return VerifyChainOutput{}, err
		}
		for _, entry := range batch {
			out.Entries++
			if out.Intact && !entry.Intact(prev) {
				out.Intact = false
				out.FirstBrokenSeq = entry.Seq
			}
			prev = entry.Hash
			afterSeq = entry.Seq
		}
		if len(batch) < verifyBatch {
			return out, nil
		}
	}
}
