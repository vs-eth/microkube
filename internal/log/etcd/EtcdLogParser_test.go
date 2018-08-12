package etcd

import "testing"

func TestInfoMessage(t *testing.T) {
	testStr := "2018-08-12 14:13:48.437712 I | etcdserver: published {Name:default ClientURLs:[https://localhost:2379]} to cluster cdf818194e3a8c32\n"
	uut := NewETCDLogParser()
	err := uut.HandleData([]byte(testStr))
	if err != nil {
		t.Fatalf("Unexpected error: {0}", err)
	}
}

func TestInfoMessageSplit(t *testing.T) {
	testStr := "2018-08-12 14:13:48.437712 I | etcdserver: published {Name:default ClientURLs:[https://localhost:2379]} to cluster cdf818194e3a8c32\n"
	uut := NewETCDLogParser()
	// Punch in message character-by-character to catch splitting bugs
	for _, character := range testStr {
		singleChar := string(character)
		err := uut.HandleData([]byte(singleChar))
		if err != nil {
			t.Fatalf("Unexpected error: %s", err)
		}
	}
}

func TestInfoMessageSplitMultiline(t *testing.T) {
	testStr := `2018-08-12 16:18:18.718670 I | etcdmain: etcd Version: 3.3.9
2018-08-12 16:18:18.718734 I | etcdmain: Git SHA: fca8add78
2018-08-12 16:18:18.718740 I | etcdmain: Go Version: go1.10.3
2018-08-12 16:18:18.718745 I | etcdmain: Go OS/Arch: linux/amd64
`
	uut := NewETCDLogParser()
	// Punch in message character-by-character to catch splitting bugs
	for _, character := range testStr {
		singleChar := string(character)
		err := uut.HandleData([]byte(singleChar))
		if err != nil {
			t.Fatalf("Unexpected error: {0}", err)
		}
	}
}
