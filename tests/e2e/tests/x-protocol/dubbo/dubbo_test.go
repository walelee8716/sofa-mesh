package dubbo

import (
	"testing"
	"flag"
	"istio.io/istio/tests/e2e/framework"
	"istio.io/istio/pkg/log"
	"os"
	"istio.io/istio/tests/util"
	"fmt"
	"time"
	"strings"
	"path/filepath"
)

const (
	testDataDir             = "tests/e2e/tests/x-protocol/dubbo/testdata"
	yamlExtension           = "yaml"
	deploymentDir           = "platform/kube"
	routeRulesDir           = "networking"
	consumerYaml            = "dubbo-consumer"
	providerYaml            = "dubbo-provider"
	busybox                 = "busybox"
	destRule                = "destination-rule-all"
	versionRule             = "virtual-service-provider-v1"
	weightRule              = "virtual-service-provider-20-80"
	consumerHTTPPort        = "8080"
	consumerName            = "dubbo-consumer"
	providerServiceName     = "dubbo-provider"
	queryName               = "test"
	expectedResponseContent = `Hello, test (from Spring Boot dubbo e2e test)`
)

type testConfig struct {
	*framework.CommonConfig
}

var (
	tc        *testConfig
	sleepTime = flag.Int("sleep_time", 15, "sleep time")
)

func TestMain(m *testing.M) {
	flag.Parse()
	if err := framework.InitLogging(); err != nil {
		log.Error("cannot setup logging")
		os.Exit(-1)
	}
	if err := setTestConfig(); err != nil {
		log.Errorf("could not create TestConfig: %v", err)
		os.Exit(-1)
	}

	os.Exit(tc.RunTest(m))
}

func TestDubbo(t *testing.T) {
	log.Infof("Begin to start test case")

	seconds := time.Second * time.Duration(*sleepTime)

	log.Infof("sleep %s", seconds)
	time.Sleep(seconds)

	url := getConsumerTargetUrl(queryName)
	log.Infof("fetch url: %s \n", url)

	actualResponseContent, err := fetchConsumerResult(url)

	if err != nil {
		log.Errorf("%s. Error %s", fmt.Sprintf("error when fetch %s", url), err)
		os.Exit(-1)
	}

	log.Infof("response content: %s \n", actualResponseContent)

	if strings.Index(actualResponseContent, expectedResponseContent) != 0 {
		log.Errorf("%s. Error %s", fmt.Sprintf("response content is not correct, expect include %s, but %s", expectedResponseContent, actualResponseContent), err)
		os.Exit(-1)
	}
}

func setTestConfig() error {
	cc, err := framework.NewCommonConfig("dubbo_test")
	if err != nil {
		return err
	}
	tc = new(testConfig)
	tc.CommonConfig = cc

	apps := getApps()

	for i := range apps {
		tc.Kube.AppManager.AddApp(&apps[i])
	}

	return nil
}

func getApps() []framework.App {
	return []framework.App{
		{
			AppYaml:    getDeploymentPath(consumerYaml),
			KubeInject: true,
		},
		{
			AppYaml:    getDeploymentPath(providerYaml),
			KubeInject: true,
		},
		{
			AppYaml:    getDeploymentPath(busybox),
			KubeInject: false,
		},
	}
}

func getConsumerTargetUrl(queryName string) (string) {
	return fmt.Sprintf("http://127.0.0.1:%s/sayHello?name=%s", consumerHTTPPort, queryName)
}

func fetchConsumerResult(url string) (string, error) {
	namespace := tc.Kube.Namespace
	kubeConfig := tc.Kube.KubeConfig

	podName, err := util.GetPodName(namespace, "app="+consumerName, kubeConfig)
	if err != nil {
		return "", err
	}

	resp, err := util.PodExec(namespace, podName, "app", "curl --silent "+url, false, kubeConfig)

	if err != nil {
		return "", err
	}

	return resp, nil
}

func getDeploymentPath(deployment string) string {
	return util.GetResourcePath(filepath.Join(testDataDir, deploymentDir, deployment+"."+yamlExtension))
}
