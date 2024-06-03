package vaultcontroller

import (
	"context"
	"fmt"
	"time"

	"github.com/go-logr/logr"
	reconcilerinterface "github.com/nephio-project/nephio/controllers/pkg/reconcilers/reconciler-interface"
	vaultClient "github.com/nephio-project/nephio/controllers/pkg/vault-client"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"

	vaultapi "github.com/hashicorp/vault/api"
)

func init() {
	reconcilerinterface.Register("vaultcontroller", &reconciler{})
}

// reconciler reconciles a VaultJWTRole object
type reconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
	Vault  *vaultapi.Client
}

func (r *reconciler) SetupWithManager(ctx context.Context, mgr ctrl.Manager, c any) (map[schema.GroupVersionKind]chan event.GenericEvent, error) {

	r.Client = mgr.GetClient()

	return nil, ctrl.NewControllerManagedBy(mgr).
		Named("VaultController").
		For(&vaultClient.VaultJWTRole{}).
		Complete(r)

}

// +kubebuilder:rbac:groups=vault.example.com,resources=vaultjwtroles,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=vault.example.com,resources=vaultjwtroles/status,verbs=get;update;patch

func (r *reconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {

	fmt.Println("WOOH TEST")
	log := r.Log.WithValues("vaultjwtrole", req.NamespacedName)

	var vaultJWTRole vaultClient.VaultJWTRole
	if err := r.Get(ctx, req.NamespacedName, &vaultJWTRole); err != nil {
		log.Error(err, "unable to fetch VaultJWTRole")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	rolePath := fmt.Sprintf("auth/jwt/role/%s", vaultJWTRole.Name)
	_, err := r.Vault.Logical().Write(rolePath, map[string]interface{}{
		"role_type":       vaultJWTRole.Spec.RoleType,
		"user_claim":      vaultJWTRole.Spec.UserClaim,
		"bound_audiences": vaultJWTRole.Spec.BoundAudiences,
		"bound_subject":   vaultJWTRole.Spec.BoundSubject,
		"token_ttl":       vaultJWTRole.Spec.TokenTtl,
		"token_policies":  vaultJWTRole.Spec.TokenPolicies,
	})
	if err != nil {
		log.Error(err, "failed to create/update Vault JWT role")
		return ctrl.Result{RequeueAfter: time.Minute}, err
	}

	log.Info("Successfully reconciled VaultJWTRole")
	return ctrl.Result{}, nil
}
