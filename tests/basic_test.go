package tests

import (
	"context"
	"io/ioutil"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/vucchaid/go-scp"
	"github.com/vucchaid/go-scp/auth"
	"golang.org/x/crypto/ssh"
)

func establishConnection(t *testing.T) scp.Client {
	// Use SSH key authentication from the auth package.
	// During testing we ignore the host key, don't to that when you use this.
	clientConfig, _ := auth.PasswordKey("bram", "test", ssh.InsecureIgnoreHostKey())

	// Create a new SCP client.
	client := scp.NewClient("127.0.0.1:2244", &clientConfig)

	// Connect to the remote server.
	err := client.Connect()
	if err != nil {
		t.Fatalf("Couldn't establish a connection to the remote server: %s", err)
	}
	return client
}

// TestCopy tests the basic functionality of copying a file to the remote
// destination.
//
// It assumes that a Docker container is running an SSH server at port 2244
// that is using password authentication. It also assumes that the directory
// /data is writable within that container and is mapped to ./tmp/ within the
// directory the test is run from.
func TestCopy(t *testing.T) {
	client := establishConnection(t)
	defer client.Close()

	// Open a file we can transfer to the remote container.
	f, _ := os.Open("./data/upload_file.txt")
	defer f.Close()

	// Create a file name with exotic characters and spaces in them.
	// If this test works for this, simpler files should not be a problem.
	filename := "Exöt1ç uploaded file.txt"

	// Finaly, copy the file over.
	// Usage: CopyFile(fileReader, remotePath, permission).
	err := client.CopyFile(context.Background(), f, "/data/"+filename, "0777")
	if err != nil {
		t.Errorf("Error while copying file: %s", err)
	}

	// Read what the receiver have written to disk.
	content, err := ioutil.ReadFile("./tmp/" + filename)
	if err != nil {
		t.Errorf("Result file could not be read: %s", err)
	}

	text := string(content)
	expected := "It Works\n"
	if strings.Compare(text, expected) != 0 {
		t.Errorf("Got different text than expected, expected %q got, %q", expected, text)
	}
}

// TestCopy tests the basic functionality of copying a file to the remote
// destination.
//
// It assumes that a Docker container is running an SSH server at port 2244
// that is using password authentication. It also assumes that the directory
// /data is writable within that container and is mapped to ./tmp/ within the
// directory the test is run from.
func TestMultipleUploadsAndDownloads(t *testing.T) {
	client := establishConnection(t)
	defer client.Close()

	// Open a file we can transfer to the remote container.
	f1, _ := os.Open("./data/upload_file.txt")
	defer f1.Close()

	f2, _ := os.Open("./data/another_file.txt")
	defer f2.Close()

	// Open files to be written too from downloads
	f_download_1, _ := os.Create("./tmp/download_result_1")
	defer f_download_1.Close()
	f_download_2, _ := os.Create("./tmp/download_result_2")
	defer f_download_2.Close()

	// Create a file name with exotic characters and spaces in them.
	// If this test works for this, simpler files should not be a problem.
	remoteFilename1 := "Exöt1ç uploaded file.txt"
	remoteFilename2 := "verywow.txt"

	err := upload(&client, f1, "/data/"+remoteFilename1, "0777")
	if err != nil {
		t.Errorf("Error while copying file: %s", err)
	}

	err = upload(&client, f2, "/data/"+remoteFilename2, "0777")
	if err != nil {
		t.Errorf("Error while copying file: %s", err)
	}

	err = download(&client, f_download_1, "/data/"+remoteFilename1)
	if err != nil {
		t.Errorf("Error while downloading file: %s", err)
	}

	err = download(&client, f_download_2, "/data/"+remoteFilename2)
	if err != nil {
		t.Errorf("Error while downloading file: %s", err)
	}

	// Read what the receiver have written to disk.
	content, err := ioutil.ReadFile("./tmp/" + remoteFilename1)
	if err != nil {
		t.Errorf("Result file could not be read: %s", err)
	}

	// Read what the receiver have written to disk.
	content2, err := ioutil.ReadFile("./tmp/" + remoteFilename2)
	if err != nil {
		t.Errorf("Result file could not be read: %s", err)
	}

	download_result_1, _ := ioutil.ReadFile("./tmp/download_result_1")
	download_result_2, _ := ioutil.ReadFile("./tmp/download_result_2")

	text1 := string(content)
	expected := "It Works\n"
	if strings.Compare(text1, expected) != 0 {
		t.Errorf("Got different text than expected, expected %q got, %q", expected, text1)
	}

	text2 := string(content2)
	expected = "Here is some stuff and things.\nEven another line.\n"
	if strings.Compare(text2, expected) != 0 {
		t.Errorf("Got different text than expected, expected %q got, %q", expected, text2)
	}

	// Compare downloaded content to written content
	download_result_1_content := string(download_result_1)
	if strings.Compare(text1, download_result_1_content) != 0 {
		t.Errorf("Downloaded result different from disk: %q %q", download_result_1_content, text1)
	}

	download_result_2_content := string(download_result_2)
	if strings.Compare(text2, download_result_2_content) != 0 {
		t.Errorf("Downloaded result different from disk: %q %q", download_result_2_content, text2)
	}

}

func upload(client *scp.Client, file *os.File, remoteFilename, perm string) error {
	return client.CopyFile(context.Background(), file, remoteFilename, "0777")
}

func download(client *scp.Client, file *os.File, remotePath string) error {
	return client.CopyFromRemote(context.Background(), file, remotePath)
}

// TestDownloadFile tests the basic functionality of copying a file from the
// remote destination.
//
// It assumes that a Docker container is running an SSH server at port 2244
// that is using password authentication. It also assumes that the directory
// /data is writable within that container and is mapped to ./tmp/ within the
// directory the test is run from.
func TestDownloadFile(t *testing.T) {
	client := establishConnection(t)
	defer client.Close()

	// Open a file we can transfer to the remote container.
	f, _ := os.Open("./data/input.txt")
	defer f.Close()

	// Create a local file to write to.
	f, err := os.OpenFile("./tmp/output.txt", os.O_RDWR|os.O_CREATE, 0777)
	if err != nil {
		t.Errorf("Couldn't open the output file")
	}
	defer f.Close()

	// Use a file name with exotic characters and spaces in them.
	// If this test works for this, simpler files should not be a problem.
	err = client.CopyFromRemote(context.Background(), f, "/input/Exöt1ç download file.txt.txt")
	if err != nil {
		t.Errorf("Copy failed from remote: %s", err.Error())
	}

	content, err := ioutil.ReadFile("./tmp/output.txt")
	if err != nil {
		t.Errorf("Result file could not be read: %s", err)
	}

	text := string(content)
	expected := "It works for download!\n"
	if strings.Compare(text, expected) != 0 {
		t.Errorf("Got different text than expected, expected %q got, %q", expected, text)
	}
}

// TestTimeoutDownload tests that a timeout error is produced if the file is not copied in the given
// amount of time.
func TestTimeoutDownload(t *testing.T) {
	client := establishConnection(t)
	defer client.Close()
	client.Timeout = 1 * time.Millisecond

	// Open a file we can transfer to the remote container.
	f, _ := os.Open("./data/upload_file.txt")
	defer f.Close()

	// Create a file name with exotic characters and spaces in them.
	// If this test works for this, simpler files should not be a problem.
	filename := "Exöt1ç uploaded file.txt"

	err := client.CopyFile(context.Background(), f, "/data/"+filename, "0777")
	if err != context.DeadlineExceeded {
		t.Errorf("Expected a timeout error but got succeeded without error")
	}
}

// TestContextCancelDownload tests that a a copy is immediately cancelled if we call context.cancel()
func TestContextCancelDownload(t *testing.T) {
	client := establishConnection(t)
	defer client.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	// Open a file we can transfer to the remote container.
	f, _ := os.Open("./data/upload_file.txt")
	defer f.Close()

	// Create a file name with exotic characters and spaces in them.
	// If this test works for this, simpler files should not be a problem.
	filename := "Exöt1ç uploaded file.txt"

	err := client.CopyFile(ctx, f, "/data/"+filename, "0777")
	if err != context.Canceled {
		t.Errorf("Expected a canceled error but transfer succeeded without error")
	}
}

func TestDownloadBadLocalFilePermissions(t *testing.T) {
	client := establishConnection(t)
	defer client.Close()

	// Create a file with local bad permissions
	// This happens only on Linux
	f, err := os.OpenFile("./tmp/output_bdf.txt", os.O_CREATE, 0644)
	if err != nil {
		t.Error("Couldn't open the output file", err.Error())
	}
	defer f.Close()

	// This should not timeout and throw an error
	err = client.CopyFromRemote(context.Background(), f, "/input/Exöt1ç download file.txt.txt")
	if err == nil {
		t.Errorf("Expected error thrown. Got nil")
	}
}

func TestFileNotFound(t *testing.T) {
	client := establishConnection(t)
	defer client.Close()

	f, err := os.OpenFile("./tmp/output_fnf.txt", os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		t.Error("Couldn't open the output file", err.Error())
	}
	// This should throw file not found on remote
	err = client.CopyFromRemote(context.Background(), f, "/input/no_such_file.txt")
	if err == nil {
		t.Errorf("Expected error thrown. Got nil")
	}
	expected := "scp: /input/no_such_file.txt: No such file or directory\n"
	if err.Error() != expected {
		t.Errorf("Expected %v, got %v", expected, err.Error())
	}
}
