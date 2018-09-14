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
	consumerServiceHTTPPort = "8080"
	consumerServiceName     = "dubbo-consumer"
	providerServiceName     = "dubbo-provider"
	queryName               = "test"
	expectedResponseContent = `Hello, test (from Spring Boot dubbo e2e test)`
)

type testConfig struct {
	*framework.CommonConfig
}

type dubboTestError struct {
	message string
}

func (err dubboTestError) Error() string {
	return fmt.Sprint(err.message)
}

var (
	tc                *testConfig
	skipCleanTestCase = flag.Bool("skip_clean_test_case", false, "whether to skip clean test case")
	sleepTime         = flag.Int("sleep_time", 15, "sleep time")
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
	apps := getApps()

	defer func() {
		if *skipCleanTestCase {
			return
		}

		for _, app := range tc.Kube.AppManager.Apps {
			log.Infof("Clean: %s", app.AppYaml)
			err := tc.Kube.AppManager.UndeployApp(app)

			if err != nil {
				log.Infof("Clean: %s failed", app.AppYaml)
			} else {
				log.Infof("Clean: %s success", app.AppYaml)

			}
		}
	}()

	for _, app := range apps {
		log.Infof("Deploy app: %+v", app)

		if err := tc.Kube.AppManager.DeployApp(&app); err != nil {
			t.Fatalf("Failed to deploy app %s: %v", app.AppYaml, err)
		}
	}

	if err := tc.Kube.AppManager.CheckDeployments(); err != nil {
		t.Fatal("Failed waiting for apps to start")
	}

	log.Infof("Begin to start test case")
	if err := validate(); err != nil {
		t.Fatalf("Failed to validate: %s", err)
	}
}

func validate() (err error) {

	seconds := time.Second * time.Duration(*sleepTime)

	log.Infof("sleep %s", seconds)
	time.Sleep(seconds)

	url := getConsumerTargetUrl(queryName)
	log.Infof("fetch url: %s \n", url)

	actualResponseContent, err := fetchConsumerResult(url)

	if err != nil {
		return dubboTestError{fmt.Sprintf("error when fetch %s : %s", url, err)}
	}

	log.Infof("response content: %s \n", actualResponseContent)

	if strings.Index(actualResponseContent, expectedResponseContent) != 0 {
		return dubboTestError{fmt.Sprintf("response content is not correct, expect include %s, but %s", expectedResponseContent, actualResponseContent)}
	}

	return nil
}

func setTestConfig() error {
	cc, err := framework.NewCommonConfig("dubbo_test")
	if err != nil {
		return err
	}
	tc = &testConfig{CommonConfig: cc}

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
	return fmt.Sprintf("http://%s.%s.svc.cluster.local:%s/sayHello?name=%s", consumerServiceName, tc.Kube.Namespace, consumerServiceHTTPPort, queryName)
}

func fetchConsumerResult(url string) (string, error) {
	namespace := tc.Kube.Namespace
	kubeConfig := tc.Kube.KubeConfig

	podName, err := util.GetPodName(namespace, "app="+busybox, kubeConfig)
	if err != nil {
		return "", err
	}

	resp, err := util.PodExec(namespace, podName, "app", "curl --silent "+url, true, kubeConfig)

	if err != nil {
		return "", err
	}

	return resp, nil
}

func getDeploymentPath(deployment string) string {
	return util.GetResourcePath(filepath.Join(testDataDir, deploymentDir, deployment+"."+yamlExtension))
}
