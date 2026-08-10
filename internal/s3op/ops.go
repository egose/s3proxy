package s3op

type Operation string

const (
	GetObject     Operation = "GetObject"
	HeadObject    Operation = "HeadObject"
	PutObject     Operation = "PutObject"
	DeleteObject  Operation = "DeleteObject"
	HeadBucket    Operation = "HeadBucket"
	ListObjectsV2 Operation = "ListObjectsV2"
	ListObjectsV1 Operation = "ListObjectsV1"
	ListBuckets   Operation = "ListBuckets"
	CopyObject    Operation = "CopyObject"
	Unknown       Operation = "Unknown"
)

func IsRead(op Operation) bool {
	switch op {
	case GetObject, HeadObject, ListObjectsV2, ListObjectsV1, ListBuckets, HeadBucket:
		return true
	}
	return false
}

func IsWrite(op Operation) bool {
	switch op {
	case PutObject, DeleteObject, CopyObject:
		return true
	}
	return false
}

func SupportsFanout(op Operation) bool {
	switch op {
	case PutObject, DeleteObject:
		return true
	}
	return false
}

func IsConfigurable(op string) bool {
	switch Operation(op) {
	case GetObject, HeadObject, PutObject, DeleteObject, HeadBucket, ListObjectsV2, ListBuckets:
		return true
	}
	return false
}

func ConfigurableOperations() []Operation {
	return []Operation{GetObject, HeadObject, PutObject, DeleteObject, HeadBucket, ListObjectsV2, ListBuckets}
}
