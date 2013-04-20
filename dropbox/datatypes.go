package dropbox

import (
	"github.com/garyburd/go-oauth/oauth"
	"net/http"
)

// 
// Represents a Dropbox Client
// 

type DropboxClient struct {
	Token  string
	Client *http.Client
	Oauth  *oauth.Client
	Creds  *oauth.Credentials
}

// 
// Represents user account information.
// 
// https://www.dropbox.com/developers/core/api#account-info
// 

type AccountInfo struct {
	Referral_link string
	Display_name  string
	Country       string
	Email         string
	Uid           uint32
	Quota_info    *QuotaInfo
}

// 
// Represents quota info for a user account.
// 

type QuotaInfo struct {
	Shared uint64
	Quota  uint64
	Normal uint64
}

//
// Represents metadata details for a response
//
// https://www.dropbox.com/developers/core/api#metadata-details
//

type DropFile struct {
	Size         string
	Rev          string
	Thumb_exists bool
	Bytes        uint64
	Modified     string
	Path         string
	Is_dir       bool
	Icon         string
	Root         string
	Mime_type    string
	Revision     uint32
	Contents     []*DropFile
}

//
// Represents a Dropbox link
//
// https://www.dropbox.com/help/167/en
//

type DropLink struct {
	Url     string
	Expires string
}

//
// Represents a copy_ref response
//
// https://www.dropbox.com/developers/core/api#copy_ref
//

type DropCopyRef struct {
	Copy_ref string
	Expires  string
}

// 
// Represents a delta response
// 
// https://www.dropbox.com/developers/core/api#delta
// 

type DropDelta struct {
	Reset    bool
	Cursor   string
	Has_more bool
	Entries  []*DropDeltaEntry
}

// 
// Individual delta entries
// 

type DropDeltaEntry struct {
	Path     string
	DropFile *DropFile
}

// 
// Wrapper interface to help with decoding DropDelta responses.
// 

type DropDeltaRaw struct {
	Reset    bool
	Cursor   string
	Has_more bool
	Entries  []interface{}
}
