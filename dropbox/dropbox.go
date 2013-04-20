package dropbox

import (
	"fmt"
	"strconv"
	"io/ioutil"
	"net/http"
	"reflect"
	"net/url"

	"github.com/garyburd/go-oauth/oauth"
)

const (
	_API_URL              = "https://api.dropbox.com/1/"
	_API_FILEOPS_ROOT_URL = _API_URL + "fileops/"
	_API_CONTENT_URL      = "https://api-content.dropbox.com/1/"
	_API_FILES_URL        = _API_CONTENT_URL + "files/sandbox/"
	_API_FILEPUT_URL      = _API_CONTENT_URL + "files_put/sandbox/"
)

var (
	oauthClient = oauth.Client{
		TemporaryCredentialRequestURI: "https://api.dropbox.com/1/oauth/request_token",
		ResourceOwnerAuthorizationURI: "https://www.dropbox.com/1/oauth/authorize",
		TokenRequestURI:               "https://api.dropbox.com/1/oauth/access_token",
	}
)

// returns a new dropbox object you can use to authenticate with and subsequently make API requests against
func NewClient(app_key string, app_secret string) *DropboxClient {
	oauthClient.Credentials = oauth.Credentials{
		Token:  app_key,
		Secret: app_secret,
	}

	return &DropboxClient{"", new(http.Client), &oauthClient, nil}
}

//
// Retrieves information about the user's account.
// 
// https://www.dropbox.com/developers/core/api#account-info
//

func (drop *DropboxClient) AccountInfo() *AccountInfo {
	info := new(AccountInfo)

	err := drop.getUrl(_API_URL+"account/info", nil, info)
	if err != nil {
		fmt.Printf("error getting account info: %v", err)
	}

	return info
}

//
// Downloads a file.
// 
// https://www.dropbox.com/developers/core/api#files-GET
//

func (drop *DropboxClient) GetFile(path string) (string, error) {
	fileAPIURL := apiContentURL(path)
	params := make(url.Values)

	drop.Oauth.SignParam(drop.Creds, "GET", fileAPIURL, params)

	res, err := http.Get(fileAPIURL + "?" + params.Encode())
	if err != nil {
		fmt.Printf("get file error %v\n", err)
		return "", err
	}
	defer res.Body.Close()

	b, _ := ioutil.ReadAll(res.Body)
	return string(b), nil
}

//
// Uploads a file using PUT semantics. This method is in most cases simpler to use than /files (POST).
// 
// https://www.dropbox.com/developers/core/api#files_put
//

func (drop *DropboxClient) PutFile(path string, body string) error {

	params := make(url.Values)
	params.Add("overwrite", "true")

	err := drop.putUrl(_API_FILEPUT_URL+path, params, body)
	if err != nil {
		fmt.Printf("error putting file: %v", err)
		return err
	}

	return nil
}

//
// Retrieves file and folder metadata.
// 
// https://www.dropbox.com/developers/core/api#metadata
//

func (drop *DropboxClient) GetFileMeta(path string) *DropFile {
	file := new(DropFile)

	err := drop.getUrl(_API_URL+"metadata/sandbox/"+path, nil, file)
	if err != nil {
		fmt.Printf("error getting file: %v", err)
	}

	return file
}

//
// A way of letting you keep up with changes to files and folders in a user's Dropbox. 
// You can periodically call /delta to get a list of "delta entries", which are instructions 
// on how to update your local state to match the server's state.
// 
// https://www.dropbox.com/developers/core/api#delta
//

func (drop *DropboxClient) Delta(cursor string) (*DropDelta, error) {
	params := make(url.Values)
	params.Add("cursor", cursor)

	raw := new(DropDeltaRaw)
	err := drop.postUrlDecode(_API_URL+"delta", params, "", &raw)
	if err != nil {
		return nil, err
	}

	delta := new(DropDelta)

	delta.Reset = raw.Reset
	delta.Cursor = raw.Cursor
	delta.Has_more = raw.Has_more

	for _, entry := range raw.Entries {
		deltaEntry := new(DropDeltaEntry)

		ref := reflect.ValueOf(entry)
		ref2 := reflect.ValueOf(ref.Index(0).Interface())
		ref3 := reflect.ValueOf(ref.Index(1).Interface())
		deltaEntry.Path = ref2.String()

		dropFile := new(DropFile)

		for _, thing := range ref3.MapKeys() {
			thing_ref := reflect.ValueOf(ref3.MapIndex(thing).Interface())
			switch thing.String() {
			case "size":
				dropFile.Size = thing_ref.String()
			case "rev":
				dropFile.Rev = thing_ref.String()
			case "icon":
				dropFile.Icon = thing_ref.String()
			case "modified":
				dropFile.Modified = thing_ref.String()
			case "is_dir":
				dropFile.Is_dir = thing_ref.Bool()
			case "thumb_exists":
				dropFile.Thumb_exists = thing_ref.Bool()
			case "bytes":
				dropFile.Bytes = uint64(thing_ref.Float())
			case "revision":
				dropFile.Revision = uint32(thing_ref.Float())
			case "mime_type":
				dropFile.Mime_type = thing_ref.String()
			case "root":
				dropFile.Root = thing_ref.String()
			case "path":
				dropFile.Path = thing_ref.String()
			}
			deltaEntry.DropFile = dropFile
		}

		delta.Entries = append(delta.Entries, deltaEntry)
	}

	return delta, nil
}

//
// Obtains metadata for the previous revisions of a file.
//
// Only revisions up to thirty days old are available (or more if the 
// Dropbox user has Pack-Rat). You can use the revision number in conjunction 
// with the /restore call to revert the file to its previous state.
// 
// https://www.dropbox.com/developers/core/api#revisions
//

func (drop *DropboxClient) Revisions(path string, revLimit int) ([]*DropFile, error) {
	params := make(url.Values)
	params.Add("rev_limit", strconv.Itoa(revLimit))

	var files []*DropFile
	err := drop.getUrl(_API_URL+"revisions/sandbox/"+path, params, &files)
	if err != nil {
		fmt.Printf("error fetching revisions for file: %v", err)
		return nil, err
	}

	return files, nil
}

//
// Restores a file path to a previous revision.
// 
// Unlike downloading a file at a given revision and then re-uploading it, 
// this call is atomic. It also saves a bunch of bandwidth.
// 
// https://www.dropbox.com/developers/core/api#restore
//

func (drop *DropboxClient) Restore(path, rev string) (*DropFile, error) {
	params := make(url.Values)
	params.Add("rev", rev)

	return drop.postUrlDecodeFile(_API_URL+"restore/sandbox/"+path, params, "")
}

//
// Returns metadata for all files and folders whose filename contains the given search string as a substring.
//
// Searches are limited to the folder path and its sub-folder hierarchy provided in the call.
// 
// https://www.dropbox.com/developers/core/api#search
//

func (drop *DropboxClient) Search(query, path string, fileLimit int, includeDeleted bool) ([]*DropFile, error) {
	params := make(url.Values)
	params.Add("query", query)
	params.Add("file_limit", strconv.Itoa(fileLimit))
	params.Add("include_deleted", strconv.FormatBool(includeDeleted))

	var files []*DropFile
	err := drop.getUrl(_API_URL+"search/sandbox/"+path, params, &files)
	if err != nil {
		fmt.Printf("error searching for file: %v", err)
		return nil, err
	}

	return files, nil
}

//
// Creates and returns a Dropbox link to files or folders users can use to view a preview of the file in a web browser.
// 
// https://www.dropbox.com/developers/core/api#shares
//

func (drop *DropboxClient) Shares(path string, shortUrl bool) (*DropLink, error) {
	params := make(url.Values)
	params.Add("short_url", strconv.FormatBool(shortUrl))

	link := new(DropLink)
	err := drop.postUrlDecode(_API_URL+"shares/sandbox/"+path, params, "", &link)
	if err != nil {
		return nil, err
	}

	return link, nil
}

//
// Returns a link directly to a file.
// 
// Similar to /shares. The difference is that this bypasses the Dropbox webserver, 
// used to provide a preview of the file, so that you can effectively stream the contents of your media.
// 
// https://www.dropbox.com/developers/core/api#media
//

func (drop *DropboxClient) Media(path string) (*DropLink, error) {
	link := new(DropLink)
	err := drop.postUrlDecode(_API_URL+"media/sandbox/"+path, nil, "", &link)
	if err != nil {
		return nil, err
	}

	return link, nil
}

//
// Creates and returns a copy_ref to a file. This reference string can be used to copy that 
// file to another user's Dropbox by passing it in as the from_copy_ref parameter on /fileops/copy.
// 
// https://www.dropbox.com/developers/core/api#copy_ref
//

func (drop *DropboxClient) CopyRef(path string) (*DropCopyRef, error) {
	copyRef := new(DropCopyRef)
	err := drop.postUrlDecode(_API_URL+"copy_ref/sandbox/"+path, nil, "", &copyRef)
	if err != nil {
		return nil, err
	}

	return copyRef, nil
}

//
// Gets a thumbnail for an image. 
// 
// Formats: jpeg (default) or png. For images that are photos, jpeg should be preferred, 
// 			while png is better for screenshots and digital art.
//
// Size: One of the following values (default: s):
//		 xs		32x32
//		 s		64x64
//		 m		128x128
//		 l		640x480
//		 xl		1024x768
// 
// https://www.dropbox.com/developers/core/api#thumbnails
//

func (drop *DropboxClient) Thumbnails(path, format, size string) ([]byte, error) {
	params := make(url.Values)
	params.Add("format", format)
	params.Add("size", size)

	reader, err := drop.postUrlToReader(_API_CONTENT_URL+"thumbnails/sandbox/"+path, params, "")
	if err != nil {
		return nil, err
	}

	return ioutil.ReadAll(reader)
}

//
// Copies a file or folder to a new location.
// 
// https://www.dropbox.com/developers/core/api#fileops-copy
//

func (drop *DropboxClient) Copy(fromPath, toPath string) (*DropFile, error) {
	params := make(url.Values)
	params.Add("root", "sandbox")
	params.Add("from_path", fromPath)
	params.Add("to_path", toPath)

	return drop.postUrlDecodeFile(_API_FILEOPS_ROOT_URL+"copy", params, "")
}

//
// Creates a folder.
// 
// https://www.dropbox.com/developers/core/api#fileops-create-folder
//

func (drop *DropboxClient) CreateFolder(path string) (*DropFile, error) {
	params := make(url.Values)
	params.Add("root", "sandbox")
	params.Add("path", path)

	return drop.postUrlDecodeFile(_API_FILEOPS_ROOT_URL+"create_folder", params, "")
}

//
// Deletes a file or folder.
// 
// https://www.dropbox.com/developers/core/api#fileops-delete
//

func (drop *DropboxClient) Delete(path string) (*DropFile, error) {
	params := make(url.Values)
	params.Add("root", "sandbox")
	params.Add("path", path)

	return drop.postUrlDecodeFile(_API_FILEOPS_ROOT_URL+"delete", params, "")
}

//
// Moves a file or folder to a new location.
// 
// https://www.dropbox.com/developers/core/api#fileops-move
//

func (drop *DropboxClient) Move(fromPath, toPath string) (*DropFile, error) {
	params := make(url.Values)
	params.Add("root", "sandbox")
	params.Add("from_path", fromPath)
	params.Add("to_path", toPath)

	return drop.postUrlDecodeFile(_API_FILEOPS_ROOT_URL+"move", params, "")
}
