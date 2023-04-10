package mock_grpc_server

import (
	"fmt"
	"io/ioutil"
	"net"
	"path/filepath"
)

type MockServer struct {
	VolumeGroupServer
}

func CreateMockServer() (*MockServer, error) {
	// Start the mock server
	tmpdir, err := tempDir()
	if err != nil {
		return nil, err
	}
	controllerServer := MockControllerServer{}
	server := newMockServer(controllerServer)
	err = server.startOnAddress("unix", filepath.Join(tmpdir, "csi.sock"))
	if err != nil {
		return nil, err
	}

	return server, nil
}

func tempDir() (string, error) {
	dir, err := ioutil.TempDir("", "volume-group-operator-test-")
	if err != nil {
		return "", fmt.Errorf("not create temporary directory", err)
	}
	return dir, nil
}

func newMockServer(vg MockControllerServer) *MockServer {
	return &MockServer{
		VolumeGroupServer: VolumeGroupServer{
			VolumeGroup: vg,
		},
	}
}

func (m *MockServer) startOnAddress(network, address string) error {
	listener, err := net.Listen(network, address)
	if err != nil {
		return err
	}

	if err := m.VolumeGroupServer.Start(listener); err != nil {
		listener.Close()
		return err
	}

	return nil
}
