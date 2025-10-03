package doppler

import (
	"context"
	"errors"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func resourceServiceAccountIdentity() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceServiceAccountIdentityCreate,
		ReadContext:   resourceServiceAccountIdentityRead,
		UpdateContext: resourceServiceAccountIdentityUpdate,
		DeleteContext: resourceServiceAccountIdentityDelete,
		Schema: map[string]*schema.Schema{
			"service_account_slug": {
				Description: "Slug of the service account",
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
			},
			"slug": {
				Description: "Slug of the service account identity",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"name": {
				Description: "The display name of the service account identity",
				Type:        schema.TypeString,
				Required:    true,
			},
			"ttl_seconds": {
				Description: "The amount of time, in seconds, that auth tokens for this identity will be valid",
				Type:        schema.TypeInt,
				Required:    true,
			},
			"config_oidc": {
				Description: "The OIDC configuration for the identity",
				Type:        schema.TypeList,
				MaxItems:    1,
				MinItems:    1,
				Required:    true,
				Elem:        &resourceServiceAccountIdentityConfigOidc,
			},
		},
	}
}

var resourceServiceAccountIdentityConfigOidc = schema.Resource{
	Schema: map[string]*schema.Schema{
		"discovery_url": {
			Description: "The public URL of the OpenID discovery service",
			Type:        schema.TypeString,
			Required:    true,
		},
		"claims_type": {
			Description: "If \"wildcard\", wildcard characters will be expanded during claims validation. Defaults to \"exact\"",
			Type:        schema.TypeString,
			Optional:    true,
			Default:     "exact",
		},
		"claims": {
			Description: "A set of valid values for a specific claim. At least \"aud\" and \"sub\" must be provided",
			Type:        schema.TypeSet,
			MinItems:    2,
			Required:    true,
			Elem:        &resourceServiceAccountIdentityConfigOidcClaims,
		},
	},
}

var resourceServiceAccountIdentityConfigOidcClaims = schema.Resource{
	Schema: map[string]*schema.Schema{
		"key": {
			Description: "The key of the claim to validate",
			Type:        schema.TypeString,
			Required:    true,
		},
		"values": {
			Description: "The set of valid values for this claim",
			Type:        schema.TypeSet,
			MinItems:    1,
			Required:    true,
			Elem: &schema.Schema{
				Type: schema.TypeString,
			},
		},
	},
}

func resourceServiceAccountIdentityCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(APIClient)

	var diags diag.Diagnostics

	serviceAccountSlug := d.Get("service_account_slug").(string)
	payload, diags := toServiceAccountIdentity(d, diags)
	if diags.HasError() {
		return diags
	}

	id, err := client.CreateServiceAccountIdentity(ctx, serviceAccountSlug, &payload)
	if err != nil {
		diags = append(diags, diag.FromErr(err)...)
		return diags
	}

	diags = updateServiceAccountIdentityState(d, serviceAccountSlug, id, diags)
	return diags
}

func resourceServiceAccountIdentityRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(APIClient)

	var diags diag.Diagnostics
	serviceAccount := d.Get("service_account_slug").(string)
	slug := d.Id()

	id, err := client.GetServiceAccountIdentity(ctx, serviceAccount, slug)
	if err != nil {
		return handleNotFoundError(err, d)
	}

	diags = updateServiceAccountIdentityState(d, serviceAccount, &id, diags)
	return diags
}

func resourceServiceAccountIdentityUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(APIClient)

	var diags diag.Diagnostics

	serviceAccountSlug := d.Get("service_account_slug").(string)
	payload, diags := toServiceAccountIdentity(d, diags)
	if diags.HasError() {
		return diags
	}

	id, err := client.UpdateServiceAccountIdentity(ctx, serviceAccountSlug, &payload)
	if err != nil {
		diags = append(diags, diag.FromErr(err)...)
		return diags
	}

	diags = updateServiceAccountIdentityState(d, serviceAccountSlug, id, diags)
	return diags
}

func resourceServiceAccountIdentityDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(APIClient)

	var diags diag.Diagnostics
	serviceAccountSlug := d.Get("service_account_slug").(string)
	slug := d.Id()

	if err := client.DeleteServiceAccountIdentity(ctx, serviceAccountSlug, slug); err != nil {
		diags = append(diags, diag.FromErr(err)...)
		return diags
	}

	return diags
}

func toServiceAccountIdentity(d *schema.ResourceData, diags diag.Diagnostics) (ServiceAccountIdentity, diag.Diagnostics) {
	id := ServiceAccountIdentity{
		Slug:       d.Id(),
		Name:       d.Get("name").(string),
		TtlSeconds: d.Get("ttl_seconds").(int),
	}

	if oidcConfigList, oidcConfigListExists := d.GetOk("config_oidc"); oidcConfigListExists {
		id.Method = "oidc"
		oidcConfig := oidcConfigList.([]interface{})[0].(map[string]interface{}) // This is required in the schema, panic if it doesn't exist
		oidcConfigClaims := make(map[string][]string)

		for _, cc := range oidcConfig["claims"].(*schema.Set).List() {
			c := cc.(map[string]interface{})
			claimValues := make([]string, 0)
			for _, cv := range c["values"].(*schema.Set).List() {
				claimValues = append(claimValues, cv.(string))
			}
			oidcConfigClaims[c["key"].(string)] = claimValues
		}
		id.ConfigOidc = ServiceAccountIdentityConfigOidc{
			DiscoveryUrl: oidcConfig["discovery_url"].(string),
			ClaimsType:   oidcConfig["claims_type"].(string),
			Claims:       oidcConfigClaims,
		}
	}

	return id, diags
}

func updateServiceAccountIdentityState(d *schema.ResourceData, serviceAccountSlug string, id *ServiceAccountIdentity, diags diag.Diagnostics) diag.Diagnostics {
	if err := d.Set("service_account_slug", serviceAccountSlug); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}

	if err := d.Set("slug", id.Slug); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}

	if err := d.Set("name", id.Name); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}

	if err := d.Set("ttl_seconds", id.TtlSeconds); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}

	switch id.Method {
	case "oidc":
		claimSet := schema.NewSet(schema.HashResource(&resourceServiceAccountIdentityConfigOidcClaims), make([]interface{}, 0))

		for k, v := range id.ConfigOidc.Claims {
			values := schema.NewSet(schema.HashString, make([]interface{}, 0))
			for _, c := range v {
				values.Add(c)
			}
			claims := map[string]interface{}{
				"key":    k,
				"values": values,
			}
			claimSet.Add(claims)
		}

		configOidc := map[string]interface{}{
			"discovery_url": id.ConfigOidc.DiscoveryUrl,
			"claims_type":   id.ConfigOidc.ClaimsType,
			"claims":        claimSet,
		}

		configOidcList := make([]map[string]interface{}, 1)
		configOidcList[0] = configOidc
		if err := d.Set("config_oidc", configOidcList); err != nil {
			diags = append(diags, diag.FromErr(err)...)
		}
	default:
		diags = append(diags, diag.FromErr(errors.New("Unknown auth method type"))...)
	}

	d.SetId(id.Slug)
	return diags
}
