package app

import (
	"posts-service/internal/db"
	pb "posts-service/proto"
)

// ToPBForTest exposes the conversion helper to cross-package tests.
func ToPBForTest(p db.Post) *pb.Post {
	return toPB(p)
}
