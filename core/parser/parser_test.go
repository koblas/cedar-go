package parser_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"

	"github.com/koblas/cedar-go/core/parser"
	"github.com/koblas/cedar-go/core/token"
)

func testRunner(t *testing.T, rules string) {
	file, err := parser.ParseFile(token.NewFileSet(), "", rules, 0)

	assert.NoError(t, err)
	assert.NotNil(t, file)
}

// func testRunnerTrace(t *testing.T, rules string) {
// 	file, err := parser.ParseFile(token.NewFileSet(), "", rules, parser.Trace)

// 	assert.NoError(t, err)
// 	assert.NotNil(t, file)
// }

func TestBasic(t *testing.T) {
	testRunner(t, `
		permit(
			principal == User::"alice",
			action == PhotoOp::"view",
			// resource
			resource in Album::"jane_vacation"
		);
	`)
}

func TestSmoke(t *testing.T) {
	testRunner(t, `permit(principal, action, resource);`)
}

func TestFunc(t *testing.T) {
	testRunner(t, `permit(principal, action, resource) when { time("now") };`)
}

func TestSimpleFail(t *testing.T) {
	rule := `
	permit(
		action == PhotoOp::"view",
		resource in Album::"jane_vacation"
	      );
	`

	// _, err := parser.ParseFile(token.NewFileSet(), "", rule, parser.Trace)
	_, err := parser.ParseFile(token.NewFileSet(), "", rule, 0)

	assert.Error(t, err)
}

// Tests from github.com/cedar-policy/cedar/blob/main/cedar-integration-tests/tests/multi
type DecimalTestSuite struct {
	suite.Suite
}

func (suite *DecimalTestSuite) TestPolicy1() {
	testRunner(suite.T(), `
permit (
	principal,
	action == Action::"view",
	resource == Photo::"VacationPhoto94.jpg"
)
when { context.confidence_score.greaterThan(decimal("0.75")) };
	`)
}

func (suite *DecimalTestSuite) TestPolicy2() {
	testRunner(suite.T(), `
permit (
	principal,
	action == Action::"view",
	resource == Photo::"VacationPhoto94.jpg"
)
when
{
	context.confidence_score.greaterThanOrEqual(decimal("0.4")) &&
	context.confidence_score.lessThanOrEqual(decimal("0.5"))
};
	`)
}

func TestDecimalSuite(t *testing.T) {
	suite.Run(t, new(DecimalTestSuite))
}

// Tests from github.com/cedar-policy/cedar/blob/main/cedar-integration-tests/tests/multi
type MultiTestSuite struct {
	suite.Suite
}

func (suite *MultiTestSuite) TestPolicy1() {
	testRunner(suite.T(), `
// This test has two Permit policies, and tests that we give the right Reason
// for each Allow
permit (
  principal == User::"alice",
  action == Action::"view",
  resource in Album::"jane_vacation"
);

permit (
  principal == User::"bob",
  action in [Action::"view", Action::"edit"],
  resource in Account::"bob"
);
	`)
}

func (suite *MultiTestSuite) TestPolicy2() {
	testRunner(suite.T(), `
// This test has a Permit, and a Forbid that partially overrides it
// Anyone can view photos in jane's "Vacation" album
permit (
  principal,
  action == Action::"view",
  resource in Album::"jane_vacation"
);

// except bob
forbid (
  principal == User::"bob",
  action == Action::"view",
  resource in Album::"jane_vacation"
);
	`)
}

func (suite *MultiTestSuite) TestPolicy3() {
	testRunner(suite.T(), `
// This test has a slightly more complicated forbid policy partially overriding
// a permit policy
// Alice's friends can view all of her photos
permit (
  principal in UserGroup::"alice_friends",
  action == Action::"view",
  resource in Account::"alice"
);

// but, as a general rule, anything marked private can only be viewed by the
// account owner
forbid (principal, action, resource)
when { resource.private }
unless { resource has account && resource.account.owner == principal };
	`)
}

func (suite *MultiTestSuite) TestPolicy4() {
	testRunner(suite.T(), `
// Multiple permit and multiple forbid policies, which could apply in various
// combinations
// Alice's friends can view all of her photos
permit (
  principal in UserGroup::"alice_friends",
  action == Action::"view",
  resource in Account::"alice"
);

// Anyone in the Sales department can view Alice's vacation photos
permit (
  principal,
  action,
  resource in Album::"alice_vacation"
)
when { principal.department == "Sales" };

// anything marked private can only be viewed by the account owner
forbid (principal, action, resource)
when { resource.private }
unless { resource has account && resource.account.owner == principal };

// deny all requests that have context.authenticated set to false
forbid (principal, action, resource)
when { !context.authenticated };
	`)
}

func (suite *MultiTestSuite) TestPolicy5() {
	testRunner(suite.T(), `
// Anyone in the Sales department can perform any action on Alice's vacation photos
permit (
  principal,
  action,
  resource in Album::"alice_vacation"
)
when { principal.department == "Sales" };

// Deny all requests that have context.authenticated set to false
forbid (principal, action, resource)
unless { context.authenticated };

// Users with job level >= 7 can view any resource
permit (
  principal,
  action == Action::"view",
  resource
)
when { principal.jobLevel >= 7 };
	`)
}

func TestMultiSuite(t *testing.T) {
	suite.Run(t, new(MultiTestSuite))
}

// -----------------------------------------------------------------------------------
// Tests from github.com/cedar-policy/cedar/blob/main/cedar-integration-tests/tests/ip
type IpTestSuite struct {
	suite.Suite
}

func (suite *IpTestSuite) TestPolicy1() {
	testRunner(suite.T(), `
	permit (
		principal,
		action == Action::"view",
		resource == Photo::"VacationPhoto94.jpg"
	      )
	      when { context.source_ip == ip("222.222.222.222") };
	`)
}

func (suite *IpTestSuite) TestPolicy2() {
	testRunner(suite.T(), `
	permit (
		principal,
		action == Action::"view",
		resource == Photo::"VacationPhoto94.jpg"
	      )
	      when
	      { !(context.source_ip.isLoopback()) && !(context.source_ip.isMulticast()) };
	`)
}

func (suite *IpTestSuite) TestPolicy3() {
	testRunner(suite.T(), `
	permit (
		principal,
		action == Action::"view",
		resource == Photo::"VacationPhoto94.jpg"
	      )
	      when { context.source_ip.isInRange(ip("222.222.222.0/24")) };
	`)
}

func TestIpSuite(t *testing.T) {
	suite.Run(t, new(IpTestSuite))
}

// -----------------------------------------------------------------------------------
// Tests from github.com/cedar-policy/cedar/blob/main/cedar-integration-tests/tests/example_use_cases_doc
type ExampleTestSuite struct {
	suite.Suite
}

func (suite *IpTestSuite) TestPolicy1a() {
	testRunner(suite.T(), `
	// Scenario 1A: One Principal, One Action, One Resource
permit (
  principal == User::"alice",
  action == Action::"view",
  resource == Photo::"VacationPhoto94.jpg"
);
	`)
}

func (suite *IpTestSuite) TestPolicy2a() {
	testRunner(suite.T(), `
	// Scenario 2A: Anyone in a given group can view a given photo
permit (
  principal in UserGroup::"jane_friends",
  action == Action::"view",
  resource == Photo::"VacationPhoto94.jpg"
);
	`)
}

func (suite *IpTestSuite) TestPolicy2b() {
	testRunner(suite.T(), `
	// Scenario 2B: Alice can view any photo in Jane's "Vacation" album
permit (
  principal == User::"alice",
  action == Action::"view",
  resource in Album::"jane_vacation"
);
	`)
}

func (suite *IpTestSuite) TestPolicy2c() {
	testRunner(suite.T(), `
	// Scenario 2C: Alice can view, edit, or comment on any photo in Jane's "Vacation" album
permit (
  principal == User::"alice",
  action in [Action::"view", Action::"edit", Action::"comment"],
  resource in Album::"jane_vacation"
);
	`)
}

func (suite *IpTestSuite) TestPolicy3a() {
	testRunner(suite.T(), `
	// Scenario 3A: Any Principal can view jane's "Vacation" album
permit (
  principal,
  action == Action::"view",
  resource in Album::"jane_vacation"
);
	`)
}

func (suite *IpTestSuite) TestPolicy3b() {
	testRunner(suite.T(), `
	// Scenario 3B: Alice can perform specified actions on any resource in Jane's account
permit (
  principal == User::"alice",
  action in [Action::"listAlbums", Action::"listPhotos", Action::"view"],
  resource in Account::"jane"
);
	`)
}

func (suite *IpTestSuite) TestPolicy3c() {
	testRunner(suite.T(), `
	// Scenario 3C: Alice can perform any action on resources in jane's "Vacation" album
permit (
  principal == User::"alice",
  action,
  resource in Album::"jane_vacation"
);
	`)
}

func (suite *IpTestSuite) TestPolicy4a() {
	testRunner(suite.T(), `
	// Scenario 4A: The album device_prototypes is viewable by anyone in the
// department HardwareEngineering with job level at least 5
permit (
  principal,
  action in [Action::"listPhotos", Action::"view"],
  resource in Album::"device_prototypes"
)
when
{ principal.department == "HardwareEngineering" && principal.jobLevel >= 5 };
	`)
}

func (suite *IpTestSuite) TestPolicy4c() {
	testRunner(suite.T(), `
	// Scenario 4C: Alice is allowed to perform any action that is "readOnly" and
// which applies to Photos or Albums. Note that this policy uses attributes on
// "action" which is not permitted by the validator.
permit (
  principal == User::"alice",
  action,
  resource
)
when { action.readOnly && ["Photo", "Album"].contains(action.appliesTo) };
	`)
}

func (suite *IpTestSuite) TestPolicy4d() {
	testRunner(suite.T(), `
	// Scenario 4D: Owners can perform any action over their own resources
//
// (slightly adapted from the doc: in our sandbox_b, resources don't have
// .owner, but they do have .account.owner)
// (but some resources, like Account itself, don't have .account, so we
// also added a "has" check)
permit (principal, action, resource)
when { resource has account && principal == resource.account.owner };
	`)
}

func (suite *IpTestSuite) TestPolicy4e() {
	testRunner(suite.T(), `
	// Scenario 4E: Anyone can view a Photo if the department attribute on the
// Principal equals the department attribute of the owner of the Photo
//
// (slightly adapted from the doc: in our sandbox_b, resources don't have
// .owner, but they do have .account.owner)
permit (
  principal,
  action == Action::"view",
  resource
)
when { principal.department == resource.account.owner.department };
	`)
}

func (suite *IpTestSuite) TestPolicy4f() {
	testRunner(suite.T(), `
// Scenario 4F: Anyone who is an owner, or an admin, can perform any action on
// the resource
//
// (slightly adapted from the doc: in our sandbox_b, resources don't have
// .owner, but they do have .account.owner)
// (but some resources, like Account itself, don't have .account, so we
// also added a "has" check)
permit (principal, action, resource)
when
{
  (resource has account && principal == resource.account.owner) ||
  resource.admins.contains(principal)
};
	`)
}

func (suite *IpTestSuite) TestPolicy5b() {
	testRunner(suite.T(), `
// Scenario 5B: Anyone can upload photos to albums in Alice's account as long as
// the photo is a JPEG or PNG with maximum size of 1MB. However, members of the
// group AVTeam can also create RAW files up to 100MB in size.
permit (
  principal,
  action == Action::"addPhoto",
  resource in Account::"alice"
)
when
{
  (["JPEG", "PNG"].contains(context.photo.filetype) &&
   context.photo.filesize_mb <= 1) ||
  (context.photo.filetype == "RAW" &&
   context.photo.filesize_mb <= 100 &&
   principal in UserGroup::"AVTeam")
};
	`)
}

func TestExampleSuite(t *testing.T) {
	suite.Run(t, new(ExampleTestSuite))
}
