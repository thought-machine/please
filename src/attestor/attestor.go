package attestor

import (
	"encoding/hex"
	"encoding/json"

	prov "github.com/in-toto/attestation/go/predicates/provenance/v1"
	attestation "github.com/in-toto/attestation/go/v1"
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
	subjects []*attestation.ResourceDescriptor
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
	externalParam := make(map[string]interface{})
	
	targetNames := make([]interface{}, 0)
	for _, v := range targets {
		targetNames = append(targetNames, v.String())
	}
	externalParam["targets"] = targetNames

	p.PbProvenance.BuildDefinition.ExternalParameters, err = structpb.NewStruct(externalParam)
	if err != nil {
		return err
	}

	// Resolved Dependencies


	// Run Details


	// Subjects
	p.subjects, err = p.Subjects(targets, state)
	if err != nil {
		return err
	}

	return nil
}

func (p *Provenance) MarshalJSON() ([]byte, error) {
	return json.Marshal(&p.PbProvenance)
}

func (p *Provenance) Subjects(targets []core.BuildLabel, state *core.BuildState) ([]*attestation.ResourceDescriptor, error) {
	subjects := []*attestation.ResourceDescriptor{}

	for _, label := range targets {
		p := state.SyncParsePackage(label)
		outputs := p.Target(label.Name).FullOutputs()

		for _, outputItem := range outputs {
			hash, err := state.PathHasher.Hash(outputItem, false, false, false)
			if err != nil {
				return nil, err
			}

			subject := &attestation.ResourceDescriptor{}
			subject.Name = outputItem
			subject.Digest = map[string]string{
				state.PathHasher.AlgoName(): hex.EncodeToString(hash),
			}

			subjects = append(subjects, subject)
		}

	}
	return subjects, nil
}