package ftpserver

import "testing"

type mockDataAddr struct {
	host string
	port int
}

func (m mockDataAddr) Host() string { return m.host }
func (m mockDataAddr) Port() int    { return m.port }
func (m mockDataAddr) Read(_ []byte) (int, error) {
	return 0, nil
}

func TestExtractRemoteAddrFromDataReader(t *testing.T) {
	got := extractRemoteAddrFromDataReader(mockDataAddr{
		host: "192.168.1.100",
		port: 12345,
	})
	if got != "192.168.1.100:12345" {
		t.Fatalf("unexpected remote addr: %s", got)
	}
}

func TestExtractRemoteAddrFromDataReaderNoPort(t *testing.T) {
	got := extractRemoteAddrFromDataReader(mockDataAddr{
		host: "10.0.0.1",
		port: 0,
	})
	if got != "10.0.0.1" {
		t.Fatalf("unexpected remote addr without port: %s", got)
	}
}
