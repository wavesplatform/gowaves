package proto

type LeaseInfo struct {
	IsActive    bool
	LeaseAmount uint64
	Recipient   WavesAddress
	Sender      WavesAddress
}
