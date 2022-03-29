## Time limited, auto-expiring group memberships for users on Google Cloud

A script in golang which demonstrates how to allow a user temporary, time-limited membership to a Google Group.  (firecall access, just in time access)

You can use this to set on-demand firecall access based on google groups.  

For example, if you need to let a specific user access to a GCP resource for a limited amount of time, you can either

- `A` Give a user IAM bindings directly to the necessary resources
- `B` Add an [IAM condition with date/time](https://cloud.google.com/iam/docs/conditions-overview#request_attributes)
- `C` Add a user to a group which has access to the resources.

The issue with `A` is you have to remember to revoke and renew access manually

With `B` you will have lingering, expired IAM conditions on the resource.  You will also have to apply the same condition to all resources that should be accessed.  IAM conditions are also limited to [certain resource types](https://cloud.google.com/iam/docs/conditions-overview#resources).  Also note the limits on IAM bindings per resource and limits on the [condition expression](https://cloud.google.com/iam/quotas#limits)


So, that leaves one option here:  create a google group that has access to resources and control the membership of that group.  A feature that makes management a lot easier is the auto-expiring group membership capability:

- [Managing membership expirations](https://cloud.google.com/identity/docs/how-to/manage-expirations)

With this, you can at least revoke access in an automated way.


The concept is certainly nothing new and there are commercial systems that do this for a living (see [CyberArk](https://www.cyberark.com/what-is/just-in-time-access/))

---

>> This repo is NOT supported by Google. caveat emptor

This sample shows how you can use the golang api to set a time-limited access control on a resource.


#### Golang

- [cloudidentity.v1beta1](https://pkg.go.dev/google.golang.org/api/cloudidentity/v1beta1)
- [groups.membership.ExpiryDetail](https://cloud.google.com/identity/docs/reference/rest/v1/groups.memberships#ExpiryDetail)


The net output should be like this:


list current members
```bash
$ date
Sat Oct 16 08:04:23 AM EDT 2021

$ gcloud identity groups memberships list --group-email=group1_3@esodemoapp2.com
---
name: groups/02grqrue4gb58m7/memberships/101638213306164197874
preferredMemberKey:
  id: user2@esodemoapp2.com
roles:
- name: MEMBER
```

First setup a project_ID to use for [quota purposes](https://cloud.google.com/sdk/gcloud/reference/auth/application-default/set-quota-project) 

```bash
export PROJECT_ID=`gcloud config get-value core/project`
export PROJECT_NUMBER=`gcloud projects describe $PROJECT_ID --format='value(projectNumber)'`
export GCLOUD_USER=`gcloud config get-value core/account`

# if you are running this as a service account, alter --member= to member="serviceAccount:$SVC_ACCOUNT_EMAIL
gcloud projects add-iam-policy-binding  $PROJECT_ID \
      --member="user:$GCLOUD_USER" 	--role='roles/serviceusage.serviceUsageConsumer'
```


apply the script to add a user for 5mins
```bash
$ go run main.go --groupID=02grqrue4gb58m7 --userID=user1@esodemoapp2.com --expireIn=5 --quotaProject=$PROJECT_ID
Member: user2@esodemoapp2.com
Added user1@esodemoapp2.com
```

confirm add
```bash
$ gcloud identity groups memberships list --group-email=group1_3@esodemoapp2.com
---
name: groups/02grqrue4gb58m7/memberships/104497032270219758212
preferredMemberKey:
  id: user1@esodemoapp2.com
roles:
- name: MEMBER
---
name: groups/02grqrue4gb58m7/memberships/101638213306164197874
preferredMemberKey:
  id: user2@esodemoapp2.com
roles:
- name: MEMBER
```

wait 5mins minutes and confirm membership is removed

```bash
$ date
Sat Oct 16 08:10:52 AM EDT 2021

$ gcloud identity groups memberships list --group-email=group1_3@esodemoapp2.com
---
name: groups/02grqrue4gb58m7/memberships/101638213306164197874
preferredMemberKey:
  id: user2@esodemoapp2.com
roles:
- name: MEMBER
```

**Note** if the user _already_ exists in the group, invoking this api will result in an error.  If you want want extend membership to an existing user, supply the set the `--autoExtend` flag

### Terraform

You could potentially use terraform as a management layer for adding/removing users.

The biggest issue with terraform auto-expiring users is that if terraform changes group membership, a different process would modify the resource which makes the terraform state out of sync. I'm keeping this here incase for documentation.

Besides, at the moment `10/16/21` the Terraform provider for [cloud_identity_group](https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/cloud_identity_group)  does NOT support the parameter to add/remove users

It should be a parameter in the magic-module definition here

[https://github.com/GoogleCloudPlatform/magic-modules/blob/master/mmv1/products/cloudidentity/api.yaml#L248](https://github.com/GoogleCloudPlatform/magic-modules/blob/master/mmv1/products/cloudidentity/api.yaml#L248)

I imagine it may look like this if this is even a legitimate thing to do with terraform...

```hcl
resource "google_cloud_identity_group_membership" "cloud_identity_group_membership_basic" {
  group = "groups/02grqrue4gb58m7"

  preferred_member_key {
    id = "user1@esodemoapp2.com"
  }
  roles {
    name = "MEMBER"
    expiry_detail {
      expire_time = "2014-10-02T15:01:23Z"
    }
  }
}
```

Terraform should also does not have support for [updating group memberships](https://cloud.google.com/identity/docs/how-to/manage-expirations#updating_the_expiration_of_a_membership).

- see [terraform-provider-google #10343](https://github.com/hashicorp/terraform-provider-google/issues/10343)

### Logging

Changes show up in Workspace Audit logs but are pretty high in latency O(mins->hrs)

The filter you can use would be something like this:

```
protoPayload.serviceName="cloudidentity.googleapis.com"
logName: "organizations/673208786098/logs/cloudaudit.googleapis.com%2Factivity"
resource.type="audited_resource"
```

which you can also view with gcloud (ofcourse...replace with your own orgID)
```bash
$ gcloud logging read  --organization=673208786098
```

Add User:

```yaml
insertId: 41616e8ca214107f662ac4cfddb7ae0c
logName: organizations/673208786098/logs/cloudaudit.googleapis.com%2Factivity
protoPayload:
  '@type': type.googleapis.com/google.cloud.audit.AuditLog
  authenticationInfo:
    principalEmail: admin@esodemoapp2.com
  authorizationInfo:
  - granted: true
    permission: cloudidentity.membership.update
    resource: cloudidentity.googleapis.com/groups/345595908567
  metadata:
    '@type': type.googleapis.com/google.cloud.audit.GroupAuditMetadata
    group: group:group1_3@esodemoapp2.com
    membershipDelta:
      member: user:user1@esodemoapp2.com
      roleDeltas:
      - action: ADD
        role: MEMBER
  methodName: google.apps.cloudidentity.groups.v1.MembershipsService.UpdateMembership
  requestMetadata:
    callerIp: 1.2.3.4
    callerSuppliedUserAgent: google-api-go-client/0.5,gzip(gfe),gzip(gfe)
  resourceName: groups/group1_3@esodemoapp2.com
  serviceName: cloudidentity.googleapis.com
receiveTimestamp: '2021-10-16T12:04:32.072691150Z'
resource:
  labels:
    method: google.apps.cloudidentity.groups.v1.MembershipsService.UpdateMembership
    service: cloudidentity.googleapis.com
  type: audited_resource
severity: NOTICE
timestamp: '2021-10-16T12:04:31.503723Z'
```

AutoRemove User: 

```yaml
insertId: bfebbcb6070346a1ff84f54fc3d7d17c
logName: organizations/673208786098/logs/cloudaudit.googleapis.com%2Factivity
protoPayload:
  '@type': type.googleapis.com/google.cloud.audit.AuditLog
  authenticationInfo:
    principalEmail: cloud-support@google.com
  authorizationInfo:
  - granted: true
    permission: cloudidentity.membership.update
    resource: cloudidentity.googleapis.com/groups/345595908567
  metadata:
    '@type': type.googleapis.com/google.cloud.audit.GroupAuditMetadata
    group: group:group1_3@esodemoapp2.com
    membershipDelta:
      member: user:user1@esodemoapp2.com
      roleDeltas:
      - action: REMOVE
        role: MEMBER
  methodName: google.apps.cloudidentity.groups.v1.MembershipsService.UpdateMembership
  requestMetadata: {}
  resourceName: groups/group1_3@esodemoapp2.com
  serviceName: cloudidentity.googleapis.com
receiveTimestamp: '2021-10-16T12:09:31.861301264Z'
resource:
  labels:
    method: google.apps.cloudidentity.groups.v1.MembershipsService.UpdateMembership
    service: cloudidentity.googleapis.com
  type: audited_resource
severity: NOTICE
timestamp: '2021-10-16T12:09:31.353801Z'
```
