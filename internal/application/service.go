package application

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"timber-stage-qualifier/internal/evidence"
	"timber-stage-qualifier/internal/repository"
)

type Clock func() time.Time
type IDGenerator func(prefix string) string

type Service struct {
	repo         *repository.SQLiteRepository
	certificates *evidence.Generator
	now          Clock
	newID        IDGenerator
}

func NewService(repo *repository.SQLiteRepository, certificates *evidence.Generator) *Service {
	return &Service{repo: repo, certificates: certificates, now: time.Now, newID: randomID}
}

func NewServiceWithDependencies(repo *repository.SQLiteRepository, certificates *evidence.Generator, now Clock, ids IDGenerator) *Service {
	return &Service{repo: repo, certificates: certificates, now: now, newID: ids}
}

func randomID(prefix string) string {
	b := make([]byte, 12)
	if _, err := rand.Read(b); err != nil {
		sum := sha256.Sum256([]byte(fmt.Sprintf("%s-%d", prefix, time.Now().UnixNano())))
		copy(b, sum[:12])
	}
	return prefix + "_" + hex.EncodeToString(b)
}

func fingerprint(command any) (string, error) {
	data, err := json.Marshal(command)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:]), nil
}

func decodeResult[T any](result repository.CommandResult) (Outcome[T], error) {
	var value T
	if err := json.Unmarshal(result.Body, &value); err != nil {
		return Outcome[T]{}, err
	}
	return Outcome[T]{Value: value, Replayed: result.Replayed}, nil
}

func (s *Service) Ready(ctx context.Context) error { return s.repo.Ready(ctx) }
