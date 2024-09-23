package resource

import (
	"context"
	"fmt"

	"github.com/spiffe/go-spiffe/v2/workloadapi"

	"github.com/spiffe/go-spiffe/v2/spiffeid"
	"github.com/spiffe/go-spiffe/v2/svid/jwtsvid"

	"sigs.k8s.io/controller-runtime/pkg/log"
)

func GetJWT(ctx context.Context) (*jwtsvid.SVID, error) {
	socketPath := "unix:///spiffe-workload-api/spire-agent.sock"
	log := log.FromContext(ctx)
	clientOptions := workloadapi.WithClientOptions(workloadapi.WithAddr(socketPath))
	jwtSource, err := workloadapi.NewJWTSource(ctx, clientOptions)
	if err != nil {
		log.Info("Unable to create JWTSource: %v", err)
	}
	defer jwtSource.Close()

	audience := "TESTING"
	spiffeID := spiffeid.RequireFromString("spiffe://example.org/nephio")

	jwtSVID, err := jwtSource.FetchJWTSVID(ctx, jwtsvid.Params{
		Audience: audience,
		Subject:  spiffeID,
	})
	if err != nil {
		log.Info("Unable to fetch JWT-SVID: %v", err)
	}

	fmt.Printf("Fetched JWT-SVID: %v\n", jwtSVID.Marshal())
	if err != nil {
		log.Error(err, "Spire auth didnt work")
	}

	return jwtSVID, err
}

type Watcher struct{}

func (Watcher) OnX509ContextUpdate(x509Context *workloadapi.X509Context) {
	fmt.Println("Update:")
	fmt.Println("  SVIDs:")
	for _, svid := range x509Context.SVIDs {
		fmt.Printf("    %s\n", svid.ID)
	}
	fmt.Println("  Bundles:")
	for _, bundle := range x509Context.Bundles.Bundles() {
		fmt.Printf("    %s (%d authorities)\n", bundle.TrustDomain(), len(bundle.X509Authorities()))
	}
}

func (Watcher) OnX509ContextWatchError(err error) {
	fmt.Println("Error:", err)
}
