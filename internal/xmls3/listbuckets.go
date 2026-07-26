package xmls3

import (
	"encoding/xml"
	"net/http"
)

type BucketEntry struct {
	Name         string `xml:"Name"`
	CreationDate string `xml:"CreationDate"`
}

type owner struct {
	ID          string `xml:"ID"`
	DisplayName string `xml:"DisplayName"`
}

type buckets struct {
	Bucket []BucketEntry `xml:"Bucket"`
}

type listAllMyBucketsResult struct {
	XMLName xml.Name `xml:"ListAllMyBucketsResult"`
	Xmlns   string   `xml:"xmlns,attr"`
	Owner   owner    `xml:"Owner"`
	Buckets buckets  `xml:"Buckets"`
}

func WriteListBuckets(w http.ResponseWriter, ownerID, ownerName string, entries []BucketEntry) {
	w.Header().Set("Content-Type", "application/xml")
	w.WriteHeader(http.StatusOK)

	result := listAllMyBucketsResult{
		Xmlns: "http://s3.amazonaws.com/doc/2006-03-01/",
		Owner: owner{
			ID:          ownerID,
			DisplayName: ownerName,
		},
		Buckets: buckets{Bucket: entries},
	}

	data, _ := xml.MarshalIndent(result, "", "  ")
	w.Write([]byte(xml.Header))
	w.Write(data)
}
