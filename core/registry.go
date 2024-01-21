package core

/*	License: GPLv3
	Authors:
		Mirko Brombin <mirko@fabricators.ltd>
		Vanilla OS Contributors <https://github.com/vanilla-os/>
	Copyright: 2024
	Description:
		ABRoot is utility which provides full immutability and
		atomicity to a Linux system, by transacting between
		two root filesystems. Updates are performed using OCI
		images, to ensure that the system is always in a
		consistent state.
*/
import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/vanilla-os/abroot/settings"
)

// A Registry instance exposes functions to interact with the configured
// Docker registry
type Registry struct {
	API string
}

// Manifest is the struct used to parse the manifest response from the registry
// it contains the manifest itself, the digest and the list of layers. This
// should be compatible with most registries, but it's not guaranteed
type Manifest struct {
	Manifest []byte
	Digest   string
	Layers   []string
}

// NewRegistry returns a new Registry instance, exposing functions to
// interact with the configured Docker registry
func NewRegistry() *Registry {
	PrintVerboseInfo("NewRegistry", "running...")
	return &Registry{
		API: fmt.Sprintf("https://%s/%s", settings.Cnf.Registry, settings.Cnf.RegistryAPIVersion),
	}
}

// HasUpdate checks if the image/tag from the registry has a different digest
// it returns the new digest and a boolean indicating if an update is available
func (r *Registry) HasUpdate(digest string) (string, bool) {
	PrintVerboseInfo("Registry.HasUpdate", "Checking for updates ...")

	token, err := GetToken()
	if err != nil {
		PrintVerboseErr("Registry.HasUpdate", 0, err)
		return "", false
	}

	manifest, err := r.GetManifest(token)
	if err != nil {
		PrintVerboseErr("Registry.HasUpdate", 1, err)
		return "", false
	}

	if manifest.Digest == digest {
		PrintVerboseInfo("Registry.HasUpdate", "no update available")
		return "", false
	}

	PrintVerboseInfo("Registry.HasUpdate", "update available. Old digest", digest, "new digest", manifest.Digest)
	return manifest.Digest, true
}

// GetToken generates a token using the provided tokenURL and returns it
func GetToken() (string, error) {
	requestUrl := fmt.Sprintf(
		"https://%s/token?scope=repository:%s:pull&service=%s",
		settings.Cnf.Registry,
		settings.Cnf.Name,
		settings.Cnf.RegistryService,
	)
	PrintVerboseInfo("Registry.GetToken", "call URI is", requestUrl)

	resp, err := http.Get(requestUrl)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("token request failed with status code: %d", resp.StatusCode)
	}

	tokenBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	// Parse token from response
	var tokenResponse struct {
		Token string `json:"token"`
	}
	err = json.Unmarshal(tokenBytes, &tokenResponse)
	if err != nil {
		return "", err
	}

	token := tokenResponse.Token
	return token, nil
}

// GetManifest returns the manifest of the image, a token is required
// to perform the request and is generated using GetToken()
func (r *Registry) GetManifest(token string) (*Manifest, error) {
	PrintVerboseInfo("Registry.GetManifest", "running...")

	manifestAPIUrl := fmt.Sprintf("%s/%s/manifests/%s", r.API, settings.Cnf.Name, settings.Cnf.Tag)
	PrintVerboseInfo("Registry.GetManifest", "call URI is", manifestAPIUrl)

	req, err := http.NewRequest("GET", manifestAPIUrl, nil)
	if err != nil {
		PrintVerboseErr("Registry.GetManifest", 0, err)
		return nil, err
	}

	req.Header.Set("User-Agent", "abroot")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		PrintVerboseErr("Registry.GetManifest", 1, err)
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		PrintVerboseErr("Registry.GetManifest", 2, err)
		return nil, err
	}

	m := make(map[string]interface{})
	err = json.Unmarshal(body, &m)
	if err != nil {
		PrintVerboseErr("Registry.GetManifest", 3, err)
		return nil, err
	}

	// digest is stored in the header
	digest := resp.Header.Get("Docker-Content-Digest")

	// we need to parse the layers to get the digests
	layers := m["layers"].([]interface{})
	var layerDigests []string
	for _, layer := range layers {
		layerDigests = append(layerDigests, layer.(map[string]interface{})["digest"].(string))
	}

	PrintVerboseInfo("Registry.GetManifest", "success")
	manifest := &Manifest{
		Manifest: body,
		Digest:   digest,
		Layers:   layerDigests,
	}

	return manifest, nil
}
