package da

type FinalizeBatch struct {
	batchIndex uint64

	l1BlockNumber uint64
}

func NewFinalizeBatch(batchIndex uint64) *FinalizeBatch {
	return &FinalizeBatch{
		batchIndex: batchIndex,
	}
}

func (f *FinalizeBatch) Type() Type {
	return FinalizeBatchType
}

func (f *FinalizeBatch) L1BlockNumber() uint64 {
	return f.l1BlockNumber
}

func (f *FinalizeBatch) BatchIndex() uint64 {
	return f.batchIndex
}

type FinalizeBatchDAV3 struct {
	BatchIndex uint64

	L1BlockNumber uint64
}

func (f *FinalizeBatch) CompareTo(other Entry) int {
	if f.BatchIndex() < other.BatchIndex() {
		return -1
	} else if f.BatchIndex() > other.BatchIndex() {
		return 1
	}
	return 0
}

func NewFinalizeBatchDAV3(batchIndex uint64) *FinalizeBatchDAV3 {
	return &FinalizeBatchDAV3{
		BatchIndex: batchIndex,
	}
}

func (f *FinalizeBatchDAV3) DAType() Type {
	return FinalizeBatchV3Type
}

func (f *FinalizeBatchDAV3) GetL1BlockNumber() uint64 {
	return f.L1BlockNumber
}
