/*
 * @Author: jason chen (jasonchen@leansoftx.com, http://smallidea.cnblogs.com)
 * @Description:
 * @Date: 2021-11
 * @LastEditors: Jason Chen
 * @LastEditTime: 2022-08-16 17:54:44
 */
package config

import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/leansoftX/smartide-cli/internal/apk/i18n"
	"github.com/leansoftX/smartide-cli/internal/model"
	"github.com/leansoftX/smartide-cli/pkg/common"
	"github.com/leansoftX/smartide-cli/pkg/docker/compose"
	"gopkg.in/yaml.v2"
	appV1 "k8s.io/api/apps/v1"
	coreV1 "k8s.io/api/core/v1"
	networkingV1 "k8s.io/api/networking/v1"
	k8sScheme "k8s.io/client-go/kubernetes/scheme"
)

// 国际化
var i18nInstance = i18n.GetInstance()

// 远程主机模式的配置文件
func NewRemoteConfig(sshRemote *common.SSHRemote, workingDir string, relativeConfigFilePath string) (
	result *SmartIdeConfig, err error) {

	if sshRemote != nil { // 从vm上加载配置文件
		// 配置文件路径
		ideYamlFilePath := common.FilePathJoin(common.OS_Linux, workingDir, relativeConfigFilePath)

		// 从远程服务器上加载配置文件
		catCommand := fmt.Sprintf(`cat %v`, ideYamlFilePath)
		configYamlContent, err := sshRemote.ExeSSHCommand(catCommand)
		if err != nil {
			return nil, err
		}
		result = newConfig(workingDir, relativeConfigFilePath, configYamlContent, OrchestratorTypeEnum_Compose, true)

		// 如果有链接文件也需要加载
		if result.IsLinkDockerComposeFile() {
			_, linkComposeFileContent := result.GetRemoteLinkDockerComposeFile(sshRemote)

			// parse
			var linkDockerCompose *compose.DockerComposeYml
			err = yaml.Unmarshal([]byte(linkComposeFileContent), &linkDockerCompose)
			if err != nil {
				return nil, err
			}
			result.Workspace.LinkCompose = linkDockerCompose

		}

	} else { // 在本地加载配置文件
		result, err = NewLocalConfig(workingDir, relativeConfigFilePath)

	}

	//TODO

	return
}

// 本地主机 模式的配置文件
func NewLocalConfig(localWorkingDir string, relativeConfigFilePath string) (
	result *SmartIdeConfig, err error) {
	// 从本地加载配置文件
	result = newConfig(localWorkingDir, relativeConfigFilePath, "", OrchestratorTypeEnum_Compose, false)

	// 如果有链接文件也需要加载
	if result.IsLinkDockerComposeFile() {
		_, linkComposeFileContent := result.GetLocalLinkDockerComposeFile()

		// parse
		var linkDockerCompose *compose.DockerComposeYml
		err = yaml.Unmarshal([]byte(linkComposeFileContent), &linkDockerCompose)
		if err != nil {
			return nil, err
		}
		result.Workspace.LinkCompose = linkDockerCompose

	}

	return
}

// k8s 模式的配置文件
func NewK8sConfig(localWorkingDir string, configFileRelativePath string) (
	result *SmartIdeK8SConfig, err error) {

	return newK8sConfig(localWorkingDir, configFileRelativePath, []string{}, "", "")
}

// k8s 模式的配置文件
func newK8sConfig(localWorkingDir string, configFileRelativePath string, linkK8sYamlRelativePaths []string,
	configContent string, k8sYamlContent string) (
	result *SmartIdeK8SConfig, err error) {

	//configFileAbsolutePath := filepath.Join(localWorkingDir, configFileRelativePath)
	//localWorkingDir := filepath.Dir(configFileAbsolutePath)
	//fileName := filepath.Base(configFileAbsolutePath)
	config := newConfig(localWorkingDir, configFileRelativePath, configContent, OrchestratorTypeEnum_K8S, false)
	result = config.ConvertToSmartIdeK8SConfig()

	parseYamlFunc := func(yamlFileContent string) error {
		for _, subYamlFileContent := range strings.Split(yamlFileContent, "---") { // 分割符
			subYamlFileContent = strings.TrimSpace(subYamlFileContent)
			if subYamlFileContent == "" {
				continue
			}

			// 遍历k8s的yaml文件
			decode := k8sScheme.Codecs.UniversalDeserializer().Decode
			obj, groupKindVersion, err := decode([]byte(subYamlFileContent), nil, nil)
			if err != nil {
				return err
			}
			if obj == nil || groupKindVersion == nil {
				return errors.New("k8s yaml 文件解析失败！")
			}

			// example: https://developers.redhat.com/blog/2020/12/16/create-a-kubernetes-operator-in-golang-to-automatically-manage-a-simple-stateful-application#set_the_controller
			switch groupKindVersion.Kind {
			case "Deployment":
				deployment := obj.(*appV1.Deployment)
				result.Workspace.Deployments = append(result.Workspace.Deployments, *deployment)
			case "Service":
				service := obj.(*coreV1.Service)
				result.Workspace.Services = append(result.Workspace.Services, *service)
			case "PersistentVolumeClaim":
				pvc := obj.(*coreV1.PersistentVolumeClaim)
				result.Workspace.PVCS = append(result.Workspace.PVCS, *pvc)
			case "NetworkPolicy":
				networkPolicy := obj.(*networkingV1.NetworkPolicy)
				result.Workspace.Networks = append(result.Workspace.Networks, *networkPolicy)
			default:
				result.Workspace.Others = append(result.Workspace.Others, obj)
			}
		}

		return nil
	}

	if k8sYamlContent != "" { // 当内容不为空的时候
		err := parseYamlFunc(k8sYamlContent)
		if err != nil {
			return nil, err
		}
	} else {
		var k8sYamlFileAbsolutePaths []string
		if len(linkK8sYamlRelativePaths) == 0 {
			//dir := path.Dir(configFileAbsolutePath)
			tempExpression := common.PathJoin(localWorkingDir, ".ide", config.Workspace.KubeDeployFileExpression) // 链接文件是相对于.ide目录的
			files, err := filepath.Glob(tempExpression)
			if err != nil {
				return nil, err
			}
			k8sYamlFileAbsolutePaths = files
		}

		for _, k8sYamlFileAbsolutePath := range k8sYamlFileAbsolutePaths {

			yamlFileBytes, err := ioutil.ReadFile(k8sYamlFileAbsolutePath)
			if err != nil {
				return nil, err
			}
			yamlFileContent := string(yamlFileBytes)
			err = parseYamlFunc(yamlFileContent)
			if err != nil {
				return nil, err
			}
		}
	}

	// 验证
	//err = result.Valid()

	return result, err
}

//
func NewComposeConfigFromContent(configFileContent string, linkComposeFileContent string) (
	result *SmartIdeConfig, linkDockerCompose *compose.DockerComposeYml, err error) {
	// 从本地加载配置文件
	result = newConfig("", "", configFileContent, OrchestratorTypeEnum_Compose, false)

	// 如果有链接文件也需要加载
	if result.IsLinkDockerComposeFile() {
		// parse
		err = yaml.Unmarshal([]byte(linkComposeFileContent), &linkDockerCompose)
		if err != nil {
			return nil, nil, err
		}
	}

	return
}

//
func NewK8sConfigFromContent(configFileContent string, linkFileContent string) (
	result *SmartIdeK8SConfig, err error) {
	return newK8sConfig("", "", []string{}, configFileContent, linkFileContent)

}

/* // 远程服务器 compose 类型配置
func NewRemoteComposeConfig(sshRemote common.SSHRemote, localWorkingDir string, configFilePath string, configContent string) (result *SmartIdeConfig) {

	return newConfig(localWorkingDir, configFilePath, configContent, OrchestratorTypeEnum_Compose, true)
}

// 本地 compose 类型配置
func NewComposeConfig(localWorkingDir string, configFilePath string) (result *SmartIdeConfig) {
	return newConfig(localWorkingDir, configFilePath, "", OrchestratorTypeEnum_Compose, false)
} */
/*
// k8s 工作区的配置
func NewK8SConfig(configFileAbsolutePath string,
	k8sYamlFileAbsolutePaths []string,
	configContent string,
	k8sYamlContent string) (result *SmartIdeK8SConfig, err error) {

	localWorkingDir := filepath.Dir(configFileAbsolutePath)
	fileName := filepath.Base(configFileAbsolutePath)
	config := newConfig(localWorkingDir, fileName, configContent, OrchestratorTypeEnum_K8S, false)
	result = config.ConvertToSmartIdeK8SConfig()

	parseYamlFunc := func(yamlFileContent string) error {
		for _, subYamlFileContent := range strings.Split(yamlFileContent, "---") { // 分割符
			subYamlFileContent = strings.TrimSpace(subYamlFileContent)
			if subYamlFileContent == "" {
				continue
			}

			// 遍历k8s的yaml文件
			decode := k8sScheme.Codecs.UniversalDeserializer().Decode
			obj, groupKindVersion, err := decode([]byte(subYamlFileContent), nil, nil)
			if err != nil {
				return err
			}
			if obj == nil || groupKindVersion == nil {
				return errors.New("k8s yaml 文件解析失败！")
			}

			// example: https://developers.redhat.com/blog/2020/12/16/create-a-kubernetes-operator-in-golang-to-automatically-manage-a-simple-stateful-application#set_the_controller
			switch groupKindVersion.Kind {
			case "Deployment":
				deployment := obj.(*appV1.Deployment)
				result.Workspace.Deployments = append(result.Workspace.Deployments, *deployment)
			case "Service":
				service := obj.(*coreV1.Service)
				result.Workspace.Services = append(result.Workspace.Services, *service)
			case "PersistentVolumeClaim":
				pvc := obj.(*coreV1.PersistentVolumeClaim)
				result.Workspace.PVCS = append(result.Workspace.PVCS, *pvc)
			case "NetworkPolicy":
				networkPolicy := obj.(*networkingV1.NetworkPolicy)
				result.Workspace.Networks = append(result.Workspace.Networks, *networkPolicy)
			default:
				result.Workspace.Others = append(result.Workspace.Others, obj)
			}
		}

		return nil
	}

	if k8sYamlContent != "" {
		err := parseYamlFunc(k8sYamlContent)
		if err != nil {
			return nil, err
		}
	} else {
		if len(k8sYamlFileAbsolutePaths) == 0 {
			dir := path.Dir(configFileAbsolutePath)
			tempExpression := common.PathJoin(dir, config.Workspace.KubeDeployFiles)
			files, err := filepath.Glob(tempExpression)
			if err != nil {
				return nil, err
			}
			k8sYamlFileAbsolutePaths = files
		}

		for _, k8sYamlFileAbsolutePath := range k8sYamlFileAbsolutePaths {

			yamlFileBytes, err := ioutil.ReadFile(k8sYamlFileAbsolutePath)
			if err != nil {
				return nil, err
			}
			yamlFileContent := string(yamlFileBytes)
			err = parseYamlFunc(yamlFileContent)
			if err != nil {
				return nil, err
			}
		}
	}

	// 验证
	err = result.Valid()

	return result, err
} */

/* //
// @Tags
// @Summary
// @Param data localWorkingDir string ""
// @Param data configFilePath string ""
// @Param data configContent string ""
func NewConfig(localWorkingDir string, configFilePath string, configContent string) (result *SmartIdeConfig) {
	return newConfig(localWorkingDir, configFilePath, configContent, OrchestratorTypeEnum_Allinone, false)
}
*/

//
func newConfig(localWorkingDir string, configFilePath string, configContent string,
	orchestratorType OrchestratorTypeEnum,
	isRemote bool) (
	result *SmartIdeConfig) {
	result = &SmartIdeConfig{}

	// 配置文件的路径
	if len(configFilePath) <= 0 {
		configFilePath = model.CONST_Default_ConfigRelativeFilePath
	}

	// 工作目录的路径
	if len(localWorkingDir) <= 0 {
		dirName, err := os.Getwd()
		if err != nil {
			common.SmartIDELog.Fatal(err)
		}
		localWorkingDir = dirName
	}

	// 加载配置
	if configContent != "" {
		contentBytes := []byte(configContent)
		contentBytes = bytes.Trim(contentBytes, "\x00")
		err := yaml.Unmarshal(contentBytes, &result)
		if err != nil {
			if result.IsNil() {
				common.CheckError(err)
			} else {
				common.SmartIDELog.ImportanceWithError(err)
			}
		}

	} else {
		if !isRemote { // 只有本地模式下，才会从本地加载配置文件
			result = loadConfigWithYamlFile(localWorkingDir, configFilePath)
		}

	}

	// 私有成员赋值
	result.Workspace.DevContainer.configRelativeFilePath = configFilePath
	result.Workspace.DevContainer.workingDirectoryPath = localWorkingDir

	// 配置文件的相对路径
	result.Workspace.DevContainer.configRelativeFilePath =
		strings.Replace(result.Workspace.DevContainer.configRelativeFilePath, result.Workspace.DevContainer.workingDirectoryPath, "", -1)

	// 置为空
	result.Workspace.DevContainer.bindingPorts = []PortMapInfo{}

	return result
}

// 从yaml文件中获取配置
func loadConfigWithYamlFile(workingDirectoryPath, configRelativeFilePath string) (result *SmartIdeConfig) {

	result = &SmartIdeConfig{}
	configFilePath := common.PathJoin(workingDirectoryPath, configRelativeFilePath)

	// check file exit
	if _, err := os.Stat(configFilePath); os.IsNotExist(err) {
		common.SmartIDELog.Error(err, i18nInstance.Main.Err_file_not_exit, configFilePath)
	}

	// read
	yamlFile, err := ioutil.ReadFile(configFilePath)
	common.CheckError(err)

	// parse
	err = yaml.Unmarshal(yamlFile, &result)
	common.CheckError(err)

	// 配置文件格式验证
	err = result.Valid()
	common.CheckError(err)

	return result
}
