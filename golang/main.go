package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"time"

	gcperrors "github.com/salrashid123/gcp_error_handler/golang/errors"
	"golang.org/x/net/context"
	cloudidentity "google.golang.org/api/cloudidentity/v1beta1"
	"google.golang.org/api/option"
)

const ()

var (
	groupID       = flag.String("groupID", "", "Group Parent")
	userID        = flag.String("userID", "", "User to add")
	autoExtend    = flag.Bool("autoExtend", false, "Automatically extend if the user is already a member")
	requestReason = flag.String("requestReason", "", "Request Reason to add for auditlogs")
	expireIn      = flag.Int("expireIn", 5, "Expire membership at (mins)")
	quotaProject  = flag.String("quotaProject", "", "Project to apply quota usage towards")
)

func main() {

	flag.Parse()

	if *groupID == "" || *userID == "" {
		fmt.Printf("groupID and userID must be set")
		flag.PrintDefaults()
		return
	}
	ctx := context.Background()

	// https://pkg.go.dev/google.golang.org/api/option#WithRequestReason
	// https://pkg.go.dev/google.golang.org/api/option#WithQuotaProject
	opts := []option.ClientOption{option.WithRequestReason(*requestReason)}
	if *quotaProject != "" {
		opts = append(opts, option.WithQuotaProject(*quotaProject))
	}
	cloudidentityService, err := cloudidentity.NewService(ctx, opts...)
	if err != nil {
		fmt.Printf("Error getting cloudIdentityService %v", err)
		return
	}

	parent := fmt.Sprintf("groups/%s", *groupID)

	resp, err := cloudidentityService.Groups.Memberships.List(parent).Do()
	if err != nil {
		fmt.Printf("Error Listing group members %v", err)
		return
	}
	var memberPath string
	for _, member := range resp.Memberships {
		fmt.Printf("Members: %s\n", member.MemberKey.Id)
		if member.MemberKey.Id == *userID {
			memberPath = member.Name
		}
	}
	// /// Add User
	expireAt := time.Now().UTC().Add(time.Duration(*expireIn) * time.Minute).Format(time.RFC3339)
	op, err := cloudidentityService.Groups.Memberships.Create(parent, &cloudidentity.Membership{
		MemberKey: &cloudidentity.EntityKey{
			Id: *userID,
		},
		Roles: []*cloudidentity.MembershipRole{{
			Name: "MEMBER",
			ExpiryDetail: &cloudidentity.ExpiryDetail{
				ExpireTime: expireAt,
			},
		}},
	}).Do()
	if err != nil {
		//fmt.Printf("Error Adding group member %v\n", err)
		gerr := gcperrors.New(gcperrors.Error{
			Err:         err,
			PrettyPrint: false,
		})
		e, nrr := gerr.GetGoogleAPIError()
		if nrr != nil {
			fmt.Printf("Error converting error %v", nrr)
			return
		}

		if e.Code == http.StatusConflict && *autoExtend {
			fmt.Printf("User is already member of group.  Extending membership duration\n")
			updateResponse, crr := cloudidentityService.Groups.Memberships.ModifyMembershipRoles(memberPath, &cloudidentity.ModifyMembershipRolesRequest{
				UpdateRolesParams: []*cloudidentity.UpdateMembershipRolesParams{
					{
						FieldMask: "expiry_detail.expire_time",
						MembershipRole: &cloudidentity.MembershipRole{
							Name: "MEMBER",
							ExpiryDetail: &cloudidentity.ExpiryDetail{
								ExpireTime: expireAt,
							},
						},
					},
				},
			}).Do()
			if crr != nil {
				fmt.Printf("Error updating membership %v", crr)
				return
			}
			fmt.Printf("Updated existing Membership: %s\n", updateResponse.Membership.Name)
		} else {
			fmt.Printf("Error creating group membership %v", gerr)
			return
		}

	} else {

		for {
			if op.Done {
				break
			}
			time.Sleep(1 * time.Second)
		}
		if op.Error != nil {
			fmt.Printf("Error adding group members %v", err)
			return
		}

		bt, err := op.Response.MarshalJSON()
		if err != nil {
			fmt.Printf("Error marshaling groups %v", err)
			return
		}
		var member cloudidentity.Membership
		err = json.Unmarshal(bt, &member)
		if err != nil {
			fmt.Printf("Error unmarshaling groups %v", err)
			return
		}

		fmt.Printf("Added %s\n", member.MemberKey.Id)
	}

}
