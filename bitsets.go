package simplebloom

type BitSets []byte

func NewBitSets(n uint) BitSets {
	bs := make(BitSets, n)
	return bs
}

func (bs BitSets) Set(index uint) {
	bs[index] = 1
}

func (bs BitSets) Unset(index uint) {
	bs[index] = 0
}

func (bs BitSets) IsSet(index uint) bool {
	return bs[index] == 1
}
