## Nephio Controller Manager

### Purpose of this controller
The purpose of this controller is to create a single binary for all the nephio management controllers for example specialiser controllers,
bootstrap controller, repo-controller, edge watcher etc.

### Design
nephio/operators/nephio-controller-manager operator is the operator manager, which will manage all the nephio mangement controllers. 
A reconcilers directory is present in the nephio/controllers/pkg module. All the management controller reconciler packages will be
created in /nephio/controllers/pkg/reconcilers/. The reconciler-interface packge in the nephio/controllers/pkg/reconcilers has the Reconciler interface with controller-runtime 
reconcile.Reconciler which will be implemented by all the nephio management controller reconcilers.
Reconciler has the method SetupWithManager which is implemented by all the controller reconcilers, through this method the reconcilers
are registered with the nephio-controller-manger operator manager.

### Controller reconciler registration flow
1. Define the reconciler struct and In init() function of the reconciler register with the reconciler interface,
       controllers.Register("repositories", &reconciler{})
            
2. Setup with nephio-controller-manager operator,
      func (r *reconciler) SetupWithManager(mgr ctrl.Manager, i interface{}) (map[schema.GroupVersionKind]chan event.GenericEvent, error) {
4.  Implement the reconciler controll loop,
       func (r *reconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) 
