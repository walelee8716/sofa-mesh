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
	testRetryTimes          = 5
)

type testConfig struct {
	*framework.CommonConfig
}

var (
	tc *testConfig
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

	standby := 0

	for i := 0; i <= testRetryTimes; i++ {
		time.Sleep(time.Duration(standby) * time.Second)

		url := getConsumerTargetUrl(queryName)
		log.Infof("%d time fetch url: %s \n", i, url)
		actualResponseContent, err := fetchConsumerResult(url)

		if err != nil {
			log.Errorf("error when fetch %s. Error %s", url, err)
		} else {
			log.Infof("success when fetch %s. response content: %s \n", url, actualResponseContent)

			if strings.Index(actualResponseContent, expectedResponseContent) != 0 {
				log.Errorf("response content is not correct, expect include %s, but %s", expectedResponseContent, actualResponseContent)
			} else {
				log.Infof("response content is correct: %s", actualResponseContent)
				break
			}
		}

		if i == testRetryTimes {
			t.Fatalf("has been try %d times, but never success", i)
		}

		standby += 10
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
