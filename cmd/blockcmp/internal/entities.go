package internal

import (
	"fmt"
)

type DataEntry struct {
	Key   string
	Value string
}

func (a *DataEntry) Equal(b DataEntry) bool {
	return a.Key == b.Key && a.Value == b.Value
}

func (a *DataEntry) String() string {
	return fmt.Sprintf("Key: %s; Value: %s", a.Key, a.Value)
}

type dataEntrySorter []DataEntry

func (a dataEntrySorter) Len() int      { return len(a) }
func (a dataEntrySorter) Swap(i, j int) { a[i], a[j] = a[j], a[i] }

func (a dataEntrySorter) Less(i, j int) bool {
	if a[i].Key == a[j].Key {
		return a[i].Value < a[j].Value
	}
	return a[i].Key < a[j].Key
}

type Transfer struct {
	Address string
	AssetID string
	Amount  int
}

func (a *Transfer) Equal(b Transfer) bool {
	return a.Address == b.Address && a.AssetID == b.AssetID && a.Amount == b.Amount
}

func (a *Transfer) String() string {
	return fmt.Sprintf("Address: %s; AssetID: %s; Amount: %d", a.Address, a.AssetID, a.Amount)
}

type transferSorter []Transfer

func (a transferSorter) Len() int      { return len(a) }
func (a transferSorter) Swap(i, j int) { a[i], a[j] = a[j], a[i] }

func (a transferSorter) Less(i, j int) bool {
	if a[i].Address == a[j].Address {
		if a[i].AssetID == a[j].AssetID {
			return a[i].Amount < a[j].Amount
		}
		return a[i].AssetID < a[j].AssetID
	}
	return a[i].Address < a[j].Address
}

type Issue struct {
	AssetID     string
	Name        string
	Description string
	Amount      int
	Decimals    int
	Reissuable  bool
	Script      string
	Nonce       int
}

func (a *Issue) Equal(b Issue) bool {
	return a.AssetID == b.AssetID && a.Name == b.Name && a.Description == b.Description &&
		a.Amount == b.Amount && a.Decimals == b.Decimals && a.Reissuable == b.Reissuable &&
		a.Script == b.Script && a.Nonce == b.Nonce
}

func (a *Issue) String() string {
	return fmt.Sprintf(
		"AssetID: %s; Name: %s; Description: %s; Amount: %d; Decimals: %d; Reissuable: %t; Script: %s; Nonce: %d",
		a.AssetID, a.Name, a.Description, a.Amount, a.Decimals, a.Reissuable, a.Script, a.Nonce,
	)
}

type issueSorter []Issue

func (a issueSorter) Len() int      { return len(a) }
func (a issueSorter) Swap(i, j int) { a[i], a[j] = a[j], a[i] }

func (a issueSorter) Less(i, j int) bool {
	if a[i].AssetID == a[j].AssetID {
		if a[i].Name == a[j].Name {
			if a[i].Description == a[j].Description {
				if a[i].Amount == a[j].Amount {
					if a[i].Decimals == a[j].Decimals {
						if a[i].Reissuable == a[j].Reissuable {
							if a[i].Script == a[j].Script {
								return a[i].Nonce < a[j].Nonce
							}
							return a[i].Script < a[j].Script
						}
						return !a[i].Reissuable
					}
					return a[i].Decimals < a[j].Decimals
				}
				return a[i].Amount < a[j].Amount
			}
			return a[i].Description < a[j].Description
		}
		return a[i].Name < a[j].Name
	}
	return a[i].AssetID < a[j].AssetID
}

type Reissue struct {
	AssetID    string
	Amount     int
	Reissuable bool
}

func (a *Reissue) Equal(b Reissue) bool {
	return a.AssetID == b.AssetID && a.Amount == b.Amount && a.Reissuable == b.Reissuable
}

func (a *Reissue) String() string {
	return fmt.Sprintf("AssetID: %s; Amount: %d; Reissuable: %t", a.AssetID, a.Amount, a.Reissuable)
}

type reissueSorter []Reissue

func (a reissueSorter) Len() int      { return len(a) }
func (a reissueSorter) Swap(i, j int) { a[i], a[j] = a[j], a[i] }

func (a reissueSorter) Less(i, j int) bool {
	if a[i].AssetID == a[j].AssetID {
		if a[i].Amount == a[j].Amount {
			return a[i].Reissuable != a[j].Reissuable && !a[i].Reissuable
		}
		return a[i].Amount < a[j].Amount
	}
	return a[i].AssetID < a[j].AssetID
}

type Burn struct {
	AssetID string
	Amount  int
}

func (a *Burn) Equal(b Burn) bool {
	return a.AssetID == b.AssetID && a.Amount == b.Amount
}

func (a *Burn) String() string {
	return fmt.Sprintf("AssetID: %s; Amount: %d", a.AssetID, a.Amount)
}

type burnSorter []Burn

func (a burnSorter) Len() int      { return len(a) }
func (a burnSorter) Swap(i, j int) { a[i], a[j] = a[j], a[i] }

func (a burnSorter) Less(i, j int) bool {
	if a[i].AssetID == a[j].AssetID {
		return a[i].Amount < a[j].Amount
	}
	return a[i].AssetID < a[j].AssetID
}

type Sponsorship struct {
	AssetID string
	MinFee  int
}

func (a *Sponsorship) Equal(b Sponsorship) bool {
	return a.AssetID == b.AssetID && a.MinFee == b.MinFee
}

func (a *Sponsorship) String() string {
	return fmt.Sprintf("AssetID: %s; MinFee: %d", a.AssetID, a.MinFee)
}

type sponsorshipSorter []Sponsorship

func (a sponsorshipSorter) Len() int      { return len(a) }
func (a sponsorshipSorter) Swap(i, j int) { a[i], a[j] = a[j], a[i] }

func (a sponsorshipSorter) Less(i, j int) bool {
	if a[i].AssetID == a[j].AssetID {
		return a[i].MinFee < a[j].MinFee
	}
	return a[i].AssetID < a[j].AssetID
}

type Lease struct {
	LeaseID   string
	Recipient string
	Amount    int
	Nonce     int
}

func (a *Lease) Equal(b Lease) bool {
	return a.LeaseID == b.LeaseID && a.Recipient == b.Recipient && a.Amount == b.Amount && a.Nonce == b.Nonce
}

func (a *Lease) String() string {
	return fmt.Sprintf("LeaseID: %s; Recipient: %s; Amount: %d; Nonce: %d", a.LeaseID, a.Recipient, a.Amount, a.Nonce)
}

type leaseSorter []Lease

func (a leaseSorter) Len() int      { return len(a) }
func (a leaseSorter) Swap(i, j int) { a[i], a[j] = a[j], a[i] }

func (a leaseSorter) Less(i, j int) bool {
	if a[i].LeaseID == a[j].LeaseID {
		if a[i].Recipient == a[j].Recipient {
			if a[i].Amount == a[j].Amount {
				return a[i].Nonce < a[j].Nonce
			}
			return a[i].Amount < a[j].Amount
		}
		return a[i].Recipient < a[j].Recipient
	}
	return a[i].LeaseID < a[j].LeaseID
}

type LeaseCancel struct {
	LeaseID string
}

func (a *LeaseCancel) Equal(b LeaseCancel) bool {
	return a.LeaseID == b.LeaseID
}

func (a *LeaseCancel) String() string {
	return fmt.Sprintf("LeaseID: %s", a.LeaseID)
}

type leaseCancelSorter []LeaseCancel

func (a leaseCancelSorter) Len() int           { return len(a) }
func (a leaseCancelSorter) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a leaseCancelSorter) Less(i, j int) bool { return a[i].LeaseID < a[j].LeaseID }
