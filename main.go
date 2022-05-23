package main

import (
	"compress/gzip"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/playwright-community/playwright-go"
	"github.com/xuri/excelize/v2"
	"io/ioutil"
	"net/http"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

func main() {
	key := flag.String("key", "", "搜索关键词")
	flag.Parse()
	if *key == "" {
		flag.PrintDefaults()
		return
	}
	fmt.Println("企业信息收集工具启动 ！ 冲呀冲呀")
	pid := play(*key)
	Save(pid, *key)
}

//unicode 转中文
func zhToUnicode(raw []byte) ([]byte, error) {
	str, err := strconv.Unquote(strings.Replace(strconv.Quote(string(raw)), `\\u`, `\u`, -1))
	if err != nil {
		return nil, err
	}
	return []byte(str), nil
}

//这里使用浏览器获取需要的pid
func play(key string) string {

	//这里检测是否下载浏览器 如果没有下载 就会直接下载
	err := playwright.Install()
	if err != nil {
		fmt.Println(err, "下载浏览器 错误")
		return "not found"

	}

	pw, err := playwright.Run()
	if err != nil {
		fmt.Println(err, "初始化运行浏览器错误")
		return "not found"

	}
	browser, err := pw.Chromium.Launch(playwright.BrowserTypeLaunchOptions{Headless: playwright.Bool(true), Args: []string{"--no-sandbox", "--disable-setuid-sandbox"}})

	if err != nil {
		fmt.Println(err, "启动谷歌浏览器 错误")
		return "not found"

	}
	ua := "Mozilla/5.0 (compatible; Baiduspider/2.0; +http://www.baidu.com/search/spider.html)"
	//添加ua 设置 跳过ssl验证
	newContext, err := browser.NewContext(playwright.BrowserNewContextOptions{
		IgnoreHttpsErrors: playwright.Bool(true), UserAgent: playwright.String(ua),
	})
	//添加隐藏浏览器指纹js 这里是为了让浏览器不提示指纹尽可能的少触发验证码
	err = newContext.AddInitScript(playwright.BrowserContextAddInitScriptOptions{Path: playwright.String("stealth.min.js")})
	if err != nil {
		fmt.Println(err, "创建新的上下文错误")
		return "not found"

	}
	//创建请求界面 这里所有请求都通过page 发送
	page, err := newContext.NewPage()

	if err != nil {
		fmt.Println(err, "创建页面请求错误")
		return "not found"

	}

	// 请求爱企查 "https://aiqicha.baidu.com/s?q=xxx公司&t=0"
	url := fmt.Sprintf("https://aiqicha.baidu.com/s?q=%s&t=0", key)
	_, err = page.Goto(url, playwright.PageGotoOptions{WaitUntil: playwright.WaitUntilStateNetworkidle})
	if err != nil {
		fmt.Println(err, "请求页面错误")
		return "not found"

	}
	//获取pid 正则
	titlesearch := regexp.MustCompile(`queryStr":"(.*?)"`)
	pidsearch := regexp.MustCompile(`pid":"(.*?)"`)
	content, err := page.Content()
	if err != nil {
		fmt.Println(err, "获取页面内容错误")
		return "not found"

	}
	title := titlesearch.FindStringSubmatch(content)
	pid := pidsearch.FindStringSubmatch(content)
	if len(title) > 0 && len(pid) > 0 {
		unicode, err := zhToUnicode([]byte(title[len(title)-1]))
		if err != nil {
			fmt.Println(err, "转换错误")
			return "not found"
		}

		if string(unicode) != key {
			fmt.Println("没有找到" + key + "的pid")
			return "not found"
		} else {
			fmt.Println("找到" + key + "的pid")
		}
	}
	return pid[len(pid)-1]

}

//读取header 和cookie 写入配置文件方便替换
func readcookie() map[string]interface{} {
	var header map[string]interface{}

	//读取文件
	filepath := "cookie.json"
	content, err := ioutil.ReadFile(filepath)
	if err != nil {
		fmt.Println("read file error")
		os.Exit(1)
	}
	//解析json
	err = json.Unmarshal(content, &header)
	if err != nil {
		fmt.Println("json unmarshal error")
		os.Exit(1)
	}
	return header
}

func get_url_info(pid string) map[string]string {
	//var xinxi_info map[string]interface{}
	//这里需要make 才能使用 注意！ 真菜
	xinxi_info := make(map[string]string)
	title_compile := regexp.MustCompile(`entName":"(.*?)"`)
	email_compile := regexp.MustCompile(`email":"(.*?)"`)
	telephone_compile := regexp.MustCompile(`telephone":"(.*?)"`)
	website_compile := regexp.MustCompile(`website":"(.*?)"`)

	url := fmt.Sprintf("https://aiqicha.baidu.com/company_detail_%s", pid)
	//请求
	html := get(url)

	title_byte := title_compile.FindStringSubmatch(html)
	title, err := zhToUnicode([]byte(title_byte[len(title_byte)-1]))
	if err != nil {
		fmt.Println("转换中文错误")
	}
	xinxi_info["公司名称"] = string(title)

	email := email_compile.FindStringSubmatch(html)
	if len(email) > 0 {
		email_text := email[len(email)-1]
		xinxi_info["邮箱"] = email_text
	}
	telephone := telephone_compile.FindStringSubmatch(html)
	website := website_compile.FindStringSubmatch(html)
	if len(telephone) > 0 {
		telephone_text := telephone[len(telephone)-1]
		xinxi_info["电话"] = telephone_text
	}
	if len(website) > 0 {
		website_text := website[len(website)-1]
		xinxi_info["网址"] = website_text
	}
	fmt.Println(xinxi_info)
	return xinxi_info

}

type Icpinfo_app_info_json_1 struct {
	Status int `json:"status"`
	Data   struct {
		Icpinfo struct {
			Total int `json:"total"`
			List  []struct {
				Domain   []string `json:"domain"`
				SiteName string   `json:"siteName"`
				HomeSite []string `json:"homeSite"`
				IcpNo    string   `json:"icpNo"`
			} `json:"list"`
		} `json:"icpinfo"`
	} `json:"data"`
}

type Icpinfo_app_info_json_2 struct {
	Status int `json:"status"`
	Data   struct {
		Total int `json:"total"`
		List  []struct {
			Domain   []string `json:"domain"`
			SiteName string   `json:"siteName"`
			HomeSite []string `json:"homeSite"`
			IcpNo    string   `json:"icpNo"`
		} `json:"list"`
	} `json:"data"`
}

//这里获取企业备案信息
func get_icpinfo_app_info(pid string) []map[string]string {
	// 知识产权 ajax 请求地址

	var icpinfo_app_info_json Icpinfo_app_info_json_1
	url := fmt.Sprintf("https://aiqicha.baidu.com/detail/intellectualPropertyAjax?pid=%s", pid)
	html := get(url)
	err := json.Unmarshal([]byte(html), &icpinfo_app_info_json)
	if err != nil {
		fmt.Println("json unmarshal error")
		return nil
	}
	if icpinfo_app_info_json.Status != 0 {
		fmt.Println("获取企业备案信息失败")
		return nil
	}
	//这里创建一个数组用来添加下面map 数据，然后循环写入表格
	icpinfo_list := make([]map[string]string, 0)
	for _, v := range icpinfo_app_info_json.Data.Icpinfo.List {
		icpinfo_map := make(map[string]string)
		icpinfo_map["网站名称"] = "1_" + v.SiteName
		icpinfo_map["域名"] = "2_" + strings.Join(v.Domain, "---")
		icpinfo_map["首页地址"] = "3_" + strings.Join(v.HomeSite, "---")
		icpinfo_map["备案号"] = "4_" + v.IcpNo
		icpinfo_list = append(icpinfo_list, icpinfo_map)

	}
	var icpinfo_app_info_json2 Icpinfo_app_info_json_2

	if icpinfo_app_info_json.Data.Icpinfo.Total > 10 {
		for i := 10; i < icpinfo_app_info_json.Data.Icpinfo.Total; i += 10 {
			url := fmt.Sprintf("https://aiqicha.baidu.com/detail/icpinfoAjax?pid=%s&page=%d", pid, i/10+1)
			html := get(url)
			err := json.Unmarshal([]byte(html), &icpinfo_app_info_json2)
			if err != nil {
				fmt.Println("json unmarshal error")
				return nil
			}
			if icpinfo_app_info_json2.Status != 0 {
				return nil
			}
			for _, v := range icpinfo_app_info_json2.Data.List {
				icpinfo_map := make(map[string]string)
				icpinfo_map["网站名称"] = "1_" + v.SiteName
				icpinfo_map["域名"] = "2_" + strings.Join(v.Domain, "---")
				icpinfo_map["首页地址"] = "3_" + strings.Join(v.HomeSite, "---")
				icpinfo_map["备案号"] = "4_" + v.IcpNo
				icpinfo_list = append(icpinfo_list, icpinfo_map)
			}
		}

	}
	return icpinfo_list

}

//这里获取 app信息  2022-5-23 app在网页下一页数据不限时了 登陆账号也不行 等待更新
func get_app_info(pid string) {

}

//这里获取 对外投资-企业组织架构信息
type foreignInvestment_Info_enterprise_json struct {
	Status int `json:"status"`
	Data   struct {
		Total int `json:"total"`
		List  []struct {
			EntName    string `json:"entName"`
			Logo       string `json:"logo"`
			RegCapital string `json:"regCapital"`
			RegRate    string `json:"regRate"`
			OpenStatus string `json:"openStatus"`
			Pid        string `json:"pid"`
			EntLink    string `json:"entLink"`
		} `json:"list"`
	} `json:"data"`
}

//这里获取对外投资信息
func foreignInvestment_getinfo_enterprise(pid string) []map[string]string {
	var foreignInvestment_info_enterprise_json foreignInvestment_Info_enterprise_json
	url := fmt.Sprintf("https://aiqicha.baidu.com/detail/investajax?p=1&size=10&pid=%s", pid)
	html := get(url)
	err := json.Unmarshal([]byte(html), &foreignInvestment_info_enterprise_json)
	if err != nil {
		fmt.Println("json unmarshal error")
		return nil
	}
	if foreignInvestment_info_enterprise_json.Status != 0 {
		fmt.Println("获取企业组织架构信息失败")
		return nil
	}
	//这里创建一个数组用来添加下面map 数据，然后循环写入表格
	info_enterprise_list := make([]map[string]string, 0)
	for _, v := range foreignInvestment_info_enterprise_json.Data.List {
		ent_map := make(map[string]string)
		ent_map["企业名称"] = "1_" + v.EntName
		ent_map["企业logo"] = "2_" + v.Logo
		ent_map["注册资本"] = "3_" + v.RegCapital
		ent_map["注册资本占比"] = "4_" + v.RegRate
		ent_map["企业状态"] = "5_" + v.OpenStatus
		ent_map["企业链接"] = "6_" + "https://aiqicha.baidu.com/" + v.EntLink
		info_enterprise_list = append(info_enterprise_list, ent_map)
		fmt.Println(ent_map)
	}
	//这里获取企业组织架构信息  分页
	if foreignInvestment_info_enterprise_json.Data.Total > 10 {
		for i := 10; i < foreignInvestment_info_enterprise_json.Data.Total; i += 10 {
			url := fmt.Sprintf("https://aiqicha.baidu.com/detail/investajax?p=%d&size=10&pid=%s", i/10+1, pid)
			html := get(url)
			err := json.Unmarshal([]byte(html), &foreignInvestment_info_enterprise_json)
			if err != nil {
				fmt.Println("json unmarshal error")
				return nil
			}
			if foreignInvestment_info_enterprise_json.Status != 0 {
				return nil
			}
			for _, v := range foreignInvestment_info_enterprise_json.Data.List {
				ent_map := make(map[string]string)
				ent_map["企业名称"] = "1_" + v.EntName
				ent_map["企业logo"] = "2_" + v.Logo
				ent_map["注册资本"] = "3_" + v.RegCapital
				ent_map["注册资本占比"] = "4_" + v.RegRate
				ent_map["企业状态"] = "5_" + v.OpenStatus
				ent_map["企业链接"] = "6_" + "https://aiqicha.baidu.com/" + v.EntLink
				info_enterprise_list = append(info_enterprise_list, ent_map)
				fmt.Println(ent_map)
			}
		}
	}
	return info_enterprise_list
}

type HoldingsInc_info_json struct {
	Status int `json:"status"`
	Data   struct {
		Total int `json:"total"`
		List  []struct {
			EntName    string  `json:"entName"`
			Pid        string  `json:"pid"`
			Logo       string  `json:"logo"`
			Proportion float64 `json:"proportion"`
		} `json:"list"`
	} `json:"data"`
}

//这里获取 控股公司-企业组织架构信息
func holdingsInc_get_info(pid string) []map[string]string {
	url := fmt.Sprintf("https://aiqicha.baidu.com/detail/holdsAjax?pid=%s&p=1&size=10", pid)
	html := get(url)
	var holdingsInc_info_json HoldingsInc_info_json
	err := json.Unmarshal([]byte(html), &holdingsInc_info_json)
	if err != nil {
		fmt.Println("json unmarshal error")
		return nil
	}
	if holdingsInc_info_json.Status != 0 {
		fmt.Println("获取控股公司-企业组织架构信息失败")
		return nil
	}
	//这里创建一个数组用来添加下面map 数据，然后循环写入表格
	info_list := make([]map[string]string, 0)
	for _, v := range holdingsInc_info_json.Data.List {
		ent_map := make(map[string]string)
		ent_map["企业名称"] = "1_" + v.EntName
		ent_map["企业logo"] = "2_" + v.Logo
		ent_map["控股比例"] = "3_" + strconv.FormatFloat(v.Proportion, 'f', 2, 64)
		ent_map["企业链接"] = "4_" + "https://aiqicha.baidu.com/company_detail_" + v.Pid
		info_list = append(info_list, ent_map)
		fmt.Println(ent_map)
	}
	//这里获取控股公司-企业组织架构信息  分页
	if holdingsInc_info_json.Data.Total > 10 {
		for i := 10; i < holdingsInc_info_json.Data.Total; i += 10 {
			url := fmt.Sprintf("https://aiqicha.baidu.com/detail/holdsAjax?pid=%s&p=%d&size=10", pid, i/10+1)
			html := get(url)
			err := json.Unmarshal([]byte(html), &holdingsInc_info_json)
			if err != nil {
				fmt.Println("json unmarshal error")
				return nil
			}
			if holdingsInc_info_json.Status != 0 {
				return nil

			}
			for _, v := range holdingsInc_info_json.Data.List {
				ent_map := make(map[string]string)
				ent_map["企业名称"] = "1_" + v.EntName
				ent_map["企业logo"] = "2_" + v.Logo
				ent_map["控股比例"] = "3_" + strconv.FormatFloat(v.Proportion, 'f', 2, 64)
				ent_map["企业链接"] = "4_" + "https://aiqicha.baidu.com/company_detail_" + v.Pid
				info_list = append(info_list, ent_map)
				fmt.Println(ent_map)
			}
		}
	}
	return info_list
}

type Branch_info_json struct {
	Status int `json:"status"`
	Data   struct {
		Total int `json:"total"`
		List  []struct {
			EntName    string `json:"entName"`
			Logo       string `json:"logo"`
			OpenStatus string `json:"openStatus"`
			Pid        string `json:"pid"`
			EntLink    string `json:"entLink"`
		} `json:"list"`
	} `json:"data"`
}

// 这里获取分支机构-企业组织架构信息
func branch_info_Get(pid string) []map[string]string {
	url := fmt.Sprintf("https://aiqicha.baidu.com/detail/branchajax?p=1&size=10&pid=%s", pid)
	html := get(url)
	var branch_info_json Branch_info_json
	err := json.Unmarshal([]byte(html), &branch_info_json)
	if err != nil {
		fmt.Println("json unmarshal error")
		return nil
	}
	if branch_info_json.Status != 0 {
		fmt.Println("获取分支机构-企业组织架构信息失败")
		return nil
	}
	//这里创建一个数组用来添加下面map 数据，然后循环写入表格
	info_list := make([]map[string]string, 0)
	for _, v := range branch_info_json.Data.List {
		ent_map := make(map[string]string)
		ent_map["企业名称"] = "1_" + v.EntName
		ent_map["企业logo"] = "2_" + v.Logo
		ent_map["企业状态"] = "5_" + v.OpenStatus
		ent_map["企业链接"] = "6_" + "https://aiqicha.baidu.com/" + v.EntLink
		info_list = append(info_list, ent_map)
		fmt.Println(ent_map)
	}
	//这里获取分支机构-企业组织架构信息  分页
	if branch_info_json.Data.Total > 10 {
		for i := 10; i < branch_info_json.Data.Total; i += 10 {
			url := fmt.Sprintf("https://aiqicha.baidu.com/detail/branchajax?p=%d&size=10&pid=%s", i/10+1, pid)
			html := get(url)
			err := json.Unmarshal([]byte(html), &branch_info_json)
			if err != nil {
				fmt.Println("json unmarshal error")
				return nil
			}
			if branch_info_json.Status != 0 {
				return nil
			}
			for _, v := range branch_info_json.Data.List {
				ent_map := make(map[string]string)
				ent_map["企业名称"] = "1_" + v.EntName
				ent_map["企业logo"] = "2_" + v.Logo
				ent_map["企业状态"] = "5_" + v.OpenStatus
				ent_map["企业链接"] = "6_" + "https://aiqicha.baidu.com/" + v.EntLink
				info_list = append(info_list, ent_map)
				fmt.Println(ent_map)
			}
		}

	}
	return info_list
}

//定义一个通用请求函数，传入url 就可以请求 返回html代码
func get(url string) string {
	header := readcookie()
	client := http.Client{}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		fmt.Println("http new request error")
		os.Exit(1)
	}
	for k, v := range header {
		req.Header.Set(k, v.(string))
	}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("http do error")
		os.Exit(1)
	}
	defer resp.Body.Close()
	reader, _ := gzip.NewReader(resp.Body)
	body, err := ioutil.ReadAll(reader)
	if err != nil {
		fmt.Println("read body error")
		os.Exit(1)
	}
	return string(body)
}

//这里都是保存到表格函数

func Save(pid string, name string) {
	excel := excelize.NewFile()
	Al_style, err := excel.NewStyle(`{"alignment":{
    "horizontal":"center",
    "vertical":"center",
	"wrap_text":true
	}}`)
	if err != nil {
		fmt.Println("设置样式错误")
		return
	}
	err2 := excel.SetColStyle("Sheet1", "A:F", Al_style)
	if err2 != nil {
		fmt.Println("设置样式错误")
		return
	}

	titleSlice := []interface{}{"公司名称", "电话", "网址", "邮箱"}
	_ = excel.SetSheetRow("Sheet1", "A1", &titleSlice)
	_ = excel.SetColWidth("Sheet1", "A", "A", 30)
	_ = excel.SetColWidth("Sheet1", "B", "H", 50)
	data := get_url_info(pid)
	vaule := Get_value(data)
	_ = excel.SetSheetRow("Sheet1", "A2", &vaule)
	titleicpinfo := []interface{}{"网站名称", "首页地址", "域名", "域名备案号"}
	_ = excel.SetSheetRow("Sheet1", "A6", &titleicpinfo)
	icpinfo := get_icpinfo_app_info(pid)
	for i, v := range icpinfo {
		v_info := Get_icp_app(v)
		_ = excel.SetSheetRow("Sheet1", fmt.Sprintf("A%d", i+7), &v_info)
	}

	_ = excel.NewSheet("投资公司")
	title_tz := []interface{}{"企业名称", "企业logo", "注册资本", "注册资本占比", "企业状态", "企业链接"}
	_ = excel.SetSheetRow("投资公司", "A1", &title_tz)
	_ = excel.SetColStyle("投资公司", "A:F", Al_style)
	_ = excel.SetColWidth("投资公司", "A", "A", 30)
	_ = excel.SetColWidth("投资公司", "B", "H", 50)
	enterprise := foreignInvestment_getinfo_enterprise(pid)
	for i, v := range enterprise {
		v_info := Get_value(v)
		_ = excel.SetSheetRow("投资公司", fmt.Sprintf("A%d", i+2), &v_info)
	}

	_ = excel.NewSheet("控股公司")

	title_kg := []interface{}{"企业名称", "企业logo", "控股比例", "企业链接"}
	_ = excel.SetSheetRow("控股公司", "A1", &title_kg)
	_ = excel.SetColStyle("控股公司", "A:F", Al_style)
	_ = excel.SetColWidth("控股公司", "A", "A", 30)
	_ = excel.SetColWidth("控股公司", "B", "H", 50)
	shareholder := holdingsInc_get_info(pid)
	for i, v := range shareholder {
		v_info := Get_value(v)
		_ = excel.SetSheetRow("控股公司", fmt.Sprintf("A%d", i+2), &v_info)
	}

	_ = excel.NewSheet("分支机构")

	title_fz := []interface{}{"企业名称", "企业logo", "企业状态", "企业链接"}
	_ = excel.SetSheetRow("分支机构", "A1", &title_fz)
	_ = excel.SetColStyle("分支机构", "A:F", Al_style)
	_ = excel.SetColWidth("分支机构", "A", "A", 30)
	_ = excel.SetColWidth("分支机构", "B", "H", 50)
	branch := branch_info_Get(pid)
	for i, v := range branch {
		v_info := Get_value(v)
		_ = excel.SetSheetRow("分支机构", fmt.Sprintf("A%d", i+2), &v_info)
	}

	if err := excel.SaveAs(name + ".xlsx"); err != nil {
		return
	}
	fmt.Println("执行完成保存到" + name + ".xlsx")
}

//这里 转换map 为slice 然后 保存一行
func Get_value(data map[string]string) []string {
	v := make([]string, 0, len(data))

	for _, value := range data {
		v = append(v, value)
	}
	sort.Strings(v)
	title := make([]string, 0, len(v))
	i := 1
	for _, k := range v {
		str_new := strconv.Itoa(i) + "_"
		str := strings.Replace(k, str_new, "", 1)
		i = i + 1
		title = append(title, str)
	}
	return title
}

func Get_icp_app(data map[string]string) []string {
	v := make([]string, 0, len(data))

	for _, value := range data {
		v = append(v, value)
	}
	sort.Strings(v)

	title := make([]string, 0, len(v))
	i := 1
	for _, k := range v {
		str_new := strconv.Itoa(i) + "_"
		i = i + 1
		str := strings.Replace(k, str_new, "", -1)
		if strings.Contains(str, "---") {
			domain_list := strings.Split(str, "---")
			title = append(title, strings.Join(domain_list, "\n"))
		} else {
			title = append(title, str)
		}
	}
	return title
}

func Get_icp_app_list(data []map[string]string) []string {
	v := make([]string, 0, len(data))
	for _, lists := range data {
		data_list := Get_icp_app(lists)
		v = append(v, data_list...)
	}
	return v

}
