package relocated

import (
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/pkg/errors"

	"github.com/deislabs/cnab-go/bundle"
	"github.com/docker/cnab-to-oci/relocation"
	"github.com/docker/go/canonical/json"
)

type Bundle struct {
	*bundle.Bundle
	RelocationMap relocation.ImageRelocationMap
}

const (
	BundleFilename        = "bundle.json"
	RelocationMapFilename = "relocation-map.json"
)

// FromBundle returns a RelocatedBundle with an empty relocation map.
func FromBundle(bndl *bundle.Bundle) *Bundle {
	return &Bundle{
		Bundle:        bndl,
		RelocationMap: relocation.ImageRelocationMap{},
	}
}

// BundleFromFile creates a relocated bundle based on the bundle file and relocation map.
func BundleFromFile(filename string) (*Bundle, error) {
	bndl, err := bundleJSON(filename)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to read bundle")
	}

	relocationMapFileName := filepath.Join(filepath.Dir(filename), RelocationMapFilename)
	relocationMap, err := relocationMapJSON(relocationMapFileName)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to read relocation map")
	}

	return &Bundle{
		Bundle:        bndl,
		RelocationMap: relocationMap,
	}, nil
}

// writeRelocationMap serializes the relocation map and writes it to a file as JSON.
func (b *Bundle) writeRelocationMap(dest string, mode os.FileMode) error {
	d, err := json.MarshalCanonical(b.RelocationMap)
	if err != nil {
		return err
	}
	return ioutil.WriteFile(dest, d, mode)
}

// Store a bundle with the relocation map as json files.
func (b *Bundle) Store(dir string) error {
	// store bundle.json
	path := filepath.Join(dir, BundleFilename)
	if err := b.WriteFile(path, 0644); err != nil {
		return errors.Wrapf(err, "failed to store bundle")
	}

	// store relocation map
	relocationMapPath := filepath.Join(dir, RelocationMapFilename)
	if err := b.writeRelocationMap(relocationMapPath, 0644); err != nil {
		return errors.Wrapf(err, "failed to store relocation map")
	}

	return nil
}

func bundleJSON(bundlePath string) (*bundle.Bundle, error) {
	data, err := ioutil.ReadFile(bundlePath)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to read file %s", bundlePath)
	}
	bndl, err := bundle.Unmarshal(data)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to unmarshal file %s", bundlePath)
	}
	return bndl, nil
}

func relocationMapJSON(relocationMapPath string) (relocation.ImageRelocationMap, error) {
	relocationMap := relocation.ImageRelocationMap{}
	_, err := os.Stat(relocationMapPath)
	if os.IsNotExist(err) {
		return relocationMap, nil
	}
	data, err := ioutil.ReadFile(relocationMapPath)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to read file %s", relocationMapPath)
	}
	if err := json.Unmarshal(data, &relocationMap); err != nil {
		return nil, errors.Wrapf(err, "failed to unmarshal file %s", relocationMapPath)
	}
	return relocationMap, nil
}
