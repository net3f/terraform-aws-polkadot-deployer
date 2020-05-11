package test

import (
    "io/ioutil"
    "os"
    "testing"
    "time"
    "fmt"

    "github.com/gruntwork-io/terratest/modules/k8s"
    "github.com/gruntwork-io/terratest/modules/random"
    "github.com/gruntwork-io/terratest/modules/terraform"
    "github.com/gruntwork-io/terratest/modules/test-structure"
    "github.com/stretchr/testify/assert"
)

func TestTerraformAWSPolkadotDeployer(t *testing.T) {
    t.Parallel()

    terraformDir := "../"

    // At the end of the test, run `terraform destroy` to clean up any resources that were created
    defer test_structure.RunTestStage(t, "teardown", func() {
        terraformOptions := test_structure.LoadTerraformOptions(t, terraformDir)
        terraform.Destroy(t, terraformOptions)
    })

    // Deploy infrastructure
    test_structure.RunTestStage(t, "setup", func() {
        terraformOptions := createTerraformOptions(t, terraformDir)
        test_structure.SaveTerraformOptions(t, terraformDir, terraformOptions)
        terraform.InitAndApply(t, terraformOptions)
    })

    // Validate Cluster Size
    test_structure.RunTestStage(t, "validate_node_count", func() {
        terraformOptions := test_structure.LoadTerraformOptions(t, terraformDir)
        testNodeCount(t, terraformOptions)
    })
}

func createTerraformOptions(t *testing.T, terraformDir string) (*terraform.Options) {

    // A unique ID we can use to namespace resources so we don't clash with anything already in the AWS account or
    // tests running in parallel
    uniqueID := random.UniqueId()

    // Set up expected values to be checked later
    nodeCount := 1
    clusterName := fmt.Sprintf("terratest-polkadot-deployer-%s", uniqueID)
    deploymentName := fmt.Sprintf("terratest-polkadot-deployment-%s", uniqueID)

    terraformOptions := &terraform.Options{
        TerraformDir: terraformDir,
        Vars: map[string]interface{}{
            "cluster_name":    clusterName,
            "deployment_name": deploymentName,
            "location":        "eu-west-1",
            "machine_type":    "t2.micro",
            "node_count":      nodeCount,
        },
        NoColor: true,
    }

    return terraformOptions
}

func createTempFile(t *testing.T, content []byte) (f *os.File){
    tempFile, err := ioutil.TempFile(os.TempDir(), random.UniqueId())
    if err != nil {
        t.Fatal("Cannot create temporary file", err)
    }

    if _, err = tempFile.Write(content); err != nil {
        t.Fatal("Failed to write to temporary file", err)
    }
    if err := tempFile.Close(); err != nil {
        t.Fatal(err)
    }

    return tempFile
}

func testNodeCount(t *testing.T, terraformOptions *terraform.Options) {
    // Setup the kubectl config and context
    kubeconfig := terraform.Output(t, terraformOptions, "kubeconfig")
    kubeconfigFile := createTempFile(t, []byte(kubeconfig))
    defer os.Remove(kubeconfigFile.Name())
    options := k8s.NewKubectlOptions("", kubeconfigFile.Name(), "default")

    // Test that the Node count matches the Terraform specification
    k8s.WaitUntilAllNodesReady(t, options, 20, 10*time.Second)
    nodes := k8s.GetNodes(t, options)
    assert.Equal(t, len(nodes), int(terraformOptions.Vars["node_count"].(float64)))
}
