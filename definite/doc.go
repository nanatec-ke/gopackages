// Package definite is a Go client for the Definite Assurance motor insurance API.
//
// It is the Go counterpart of the @nana-tec/definite-sdk TypeScript package and
// covers the same endpoints: client and intermediary onboarding, reference data
// (products, vehicle makes and models), premium calculation, proposals,
// payments, policy lookup and DMVIC certificate generation.
//
// # Quick start
//
//	client, err := definite.NewClient(&definite.Config{
//		Environment: definite.UAT,
//		Credentials: definite.Credentials{Username: "api-user", Password: "api-pass"},
//	})
//	if err != nil {
//		log.Fatal(err)
//	}
//
//	ctx := context.Background()
//	found, err := client.CheckClient(ctx, "33669600")
//
// Every request carries HTTP Basic authentication; there is no token to manage.
// A Client is safe for concurrent use and should be reused.
//
// # Issuance flow
//
//  1. CheckClient, then CreateClient if the client does not exist.
//  2. GetProducts and GetMakes for the product, cover option, make and model IDs.
//  3. CalculatePremium to quote.
//  4. CreateProposal. Its ProposalDetails carries OID, InsuredItem and Note.
//  5. InitiatePayment with OID as Proposal and InsuredItem.
//  6. GenerateCertificate with OID as Proposal and Note as NoteOID.
//
// # Two response shapes
//
// Most endpoints return the Response envelope — {status, message, data,
// errorCode}. CreateProposal and GenerateCertificate do not: they report their
// outcome in a boolean "success" field, with the record at the top level. The
// client normalises both, so a failure is always an error.
//
// Response fields decode leniently (see FlexInt, FlexFloat, FlexString and
// FlexBool) because the API's shapes are only partly documented. Every
// response also keeps the full body in Raw.
//
// # Errors
//
// Every method returns *ClientError. Use AsClientError, IsAuthError,
// IsTimeoutError and IsInvalidInputError, or compare ClientError.Code with the
// Err* constants. Requests are validated locally before anything is sent.
//
// # Caching
//
// GetProducts and GetMakes are cached for an hour by default. Choose in-memory
// (the default), file-backed, or no caching with Config.CacheStrategy, or plug
// in a shared backend by setting Config.Cache.
package definite
