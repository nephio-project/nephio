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
3. Implement the reconciler controll loop,
       func (r *reconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) 
### Usage
The loaded reconcilers have to be enabled, following are the ways they can be enabled,
1. set env variable, example ENABLE_REPOSITORIES=true
2. pass list of reconcilers while running the manager, example ./manager --reconcilers=repositories . 
3. --reconcilders=* will enable all the reconcilers.

### Environment Variables
For the repository and token reconciler ( copied from repository README)
#### Repository controller
Based on the environment variables we help the controller to connect to the gitea server.

A secret is required to connect to the git server with username and password. The default name and namespace are resp. `git-user-secret ` and POD_NAMESPACE where the token controller runs.
With the following environment variable the defaults can be changed:
- GIT_SECRET_NAME: sets the name of the secret to connect to the git server
- GIT_NAMESPACE: sets the namespace where to find the secret to connect to the git server

The URL to connect to the git server is provided through an environment variable. This is a mandatory environment variable

- GIT_URL = https://172.18.0.200:3000

#### IPAM and VLAN specializer
- CLIENT_PROXY_ADDRESS
