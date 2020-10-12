package bls

// type Hasher interface {
// 	Hash(message *Message) (*PointG1, error)
// 	UnsafeMode()
// }

// type Mapper interface {
// 	mapTo(input []byte) (*PointG1, error)
// 	unsafeMode()
// }

// type TIMapper struct {
// 	ec *bn254.G1
// }

// type FTMapper struct {
// 	ec *bn254.G1
// }

// type HasherSHA256 struct {
// 	mapper Mapper
// }

// type HasherKeccak256 struct {
// 	mapper Mapper
// }

// func NewHasher_SHA256_TI() Hasher {
// 	mapper := &TIMapper{}
// 	return &HasherSHA256{mapper}
// }

// func NewHasher_SHA256_FT() Hasher {
// 	mapper := &FTMapper{}
// 	return &HasherSHA256{mapper}
// }

// func NewHasher_Keccak_TI() Hasher {
// 	mapper := &TIMapper{}
// 	return &HasherKeccak256{mapper}
// }

// func NewHasher_Keccak_FT() Hasher {
// 	mapper := &FTMapper{}
// 	return &HasherKeccak256{mapper}
// }

// func (h *HasherSHA256) Hash(message *Message) (*PointG1, error) {
// 	H := sha256.New()
// 	_, _ = H.Write(message.Domain)
// 	_, _ = H.Write(message.Message)
// 	digest := H.Sum(nil)
// 	return h.mapper.mapTo(digest)
// }

// func (h *HasherSHA256) UnsafeMode() {
// 	h.mapper.unsafeMode()
// }

// func (h *HasherKeccak256) Hash(message *Message) (*PointG1, error) {
// 	digest := crypto.Keccak256(message.Domain, message.Message)
// 	return h.mapper.mapTo(digest)
// }

// func (h *HasherKeccak256) UnsafeMode() {
// 	h.mapper.unsafeMode()
// }

// func (m *TIMapper) mapTo(input []byte) (*PointG1, error) {
// 	ec := m.ec
// 	if ec == nil {
// 		ec = bn254.NewG1()
// 	}
// 	return ec.MapToPointTI(input)
// }

// func (m *TIMapper) unsafeMode() {
// 	if m.ec == nil {
// 		m.ec = bn254.NewG1()
// 	}
// }

// func (m *FTMapper) mapTo(input []byte) (*PointG1, error) {
// 	ec := m.ec
// 	if ec == nil {
// 		ec = bn254.NewG1()
// 	}
// 	return ec.MapToPointFT(input)
// }

// func (m *FTMapper) unsafeMode() {
// 	if m.ec == nil {
// 		m.ec = bn254.NewG1()
// 	}
// }
