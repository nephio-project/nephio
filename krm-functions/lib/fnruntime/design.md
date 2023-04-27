# fn-runtime

The fn-runtime project is a set of go libraries for building Kpt functions/controllers that are part of a "`Condition` choreography" also called a "`Condition` dance.

The "`Condition` choreography/dance" is a set of kpt function and controllers that perform independent actions on a kpt package. Together and in harmony a declarative outcome is achieved on a kpt package (a set of KRM resources).

When comparing the conditional dance with a regular Kubernetes controller built upon the controller runtime, we can make a number of analogies.
- The event driven nature on a For object in the controller runtime is taken by Porch, which periodically presents the functions/controllers with the latest status of the package. The fn/controllers provide an idempotent operation on the package based on the resource presented.
- The fn runtime presents For/Own/Watch resources that have a similar meaning in the fn-runtime as the controller runtime
    - For: the parent KRM resource the fn-runtime operates on
    - Own: child resources which lifecycle is influenced by the parent resource or the parent resource depends upon
    - Watch: extra resources relevant for the fn to operate on
- Garbage collecion/owner reference is implemented in the fn-runtime using an annotation with owner key
- Delete and finalizers are implemented using an annotation and the delete operation is always performed by the downstream function.

Conditions are used to signal work from one fn/controller to another and are acted upon in a choreography/dance.

The fn runtime splits the implementation is two distinct runtimes. The upstream fn-runtime and downstream fn-runtime. 

## upstream fn-runtime

An upstream fn-runtime provides lifecycle on child resources (owns) based on a parent resource (for)

Adjacent resources (watch) can be used to influence the behavior the function executes. For example, if a relevant resource is missing it can trigger a garbage collection operation which deletes all resources based on a certain owner reference.

In the upstream fn-runtime we implemented 2 types of child resources (owns); upstream and downstream child resources.
- Upstream child resources have all the knowledge to perform a full lifecycle (CRUD) of the resource. Create and Updates are performed in the upstream fn/controller
- Downstream child resources rely on other information before a full lifecycle can be performed. Create and updates are performed in the downstream fn/controller.

To provide a consistent behavior deletes are always implemented based on the finalizer approach. The downstream fn/controller implements the delete to ensure remote resources are cleaned up properly.

## downstream fn-runtime

The downstream fn-runtime provides lifecycle on a parent (for) resource and leverages dependent (own) KRM resources to be able to execute create/update operations.

As with upstream fn-runtime adjacent (watch) resources information can be used to assist the downstream fn/controller.

The downstream fn-runtime updates/creates the resource (for) if all dependent conditions are met. A delete operation on the parent operation is done if not all conditions are ready or if the condition annotation is signalled.

There are 2 types of downstream runtime usages, which are called specific downstream and wildcard downstream fn/controllers.

The wildcard has no own/dependent resource since all resources are relevant for the lifecycle of the parent resource. A wildcard is typically used as the final function/controller in the "`Condition` choreography/dance".

A specific downstream implementation is working an a specific set of dependent resource that influence the lifecycle of the parent resource.