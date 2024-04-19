package attestor

import (
	"encoding/json"

	prov "github.com/in-toto/attestation/go/predicates/provenance/v1"
	// v1 "github.com/in-toto/attestation/go/v1"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/thought-machine/please/src/cli"
	"github.com/thought-machine/please/src/core"
)

const (
	Name             = "please-build"
	Type             = "https://slsa.dev/provenance/v1.0"
	BuildType        = "https://please.build/buildtypes/build@v0.1"
	DefaultBuilderId = "https://please.build/please-build@v0.1"
)

type Provenance struct {
	PbProvenance prov.Provenance
	// products     map[string]string
	// subjects     map[string]string
	// export       bool
}

func New() *Provenance {
	return &Provenance{}
}

func (p *Provenance) Name() string {
	return Name
}

func (p *Provenance) Type() string {
	return Type
}

func (p *Provenance) Attest(targets, preTargets []core.BuildLabel, state *core.BuildState, config *core.Configuration, arch cli.Arch) error {
	builder := prov.Builder{}
	metadata := prov.BuildMetadata{}
	p.PbProvenance.BuildDefinition = &prov.BuildDefinition{}
	p.PbProvenance.RunDetails = &prov.RunDetails{Builder: &builder, Metadata: &metadata}

	p.PbProvenance.BuildDefinition.BuildType = BuildType
	p.PbProvenance.RunDetails.Builder.Id = DefaultBuilderId

	// Internal Parameters
	internalParam := make(map[string]interface{})
	internalParam["version"] = config.Please.Version.VersionString()

	var err error
	p.PbProvenance.BuildDefinition.InternalParameters, err = structpb.NewStruct(internalParam)
	if err != nil {
		return err
	}

	// External Parameters


	// Resolved Dependencies


	// Run Details


	// Subjects

	return nil
}

func (p *Provenance) MarshalJSON() ([]byte, error) {
	return json.Marshal(&p.PbProvenance)
}
