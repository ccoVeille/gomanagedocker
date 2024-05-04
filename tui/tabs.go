package tui

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/ajayd-san/gomanagedocker/dockercmd"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type tabId int

const (
	images tabId = iota
	containers
	volumes
)

type Model struct {
	dockerClient dockercmd.DockerClient
	Tabs         []string
	TabContent   []listModel
	activeTab    int
	width        int
	height       int
	logsPager    PagerModel
	showLogs     bool
}

type TickMsg time.Time

func doTick() tea.Cmd {
	return tea.Tick(500*time.Millisecond, func(t time.Time) tea.Msg { return TickMsg(t) })
}

func (m Model) Init() tea.Cmd {
	return doTick()
}

func NewModel(tabs []string) Model {
	contents := make([]listModel, 3)

	for i, tabKind := range []tabId{images, containers, volumes} {
		contents[i] = InitList(tabKind)
	}

	return Model{
		dockerClient: dockercmd.NewDockerClient(),
		Tabs:         tabs,
		TabContent:   contents,
		logsPager:    PagerModel{content: string("idk")},
	}
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		windowStyle = windowStyle.
			Width(m.width - listDocStyle.GetHorizontalFrameSize() - 2).
			Height(m.height - listDocStyle.GetVerticalFrameSize() - 3)

		//change list dimentions when window size changes
		// TODO: change width
		for index := range m.TabContent {
			m.getList(index).SetWidth(msg.Width)
			m.getList(index).SetHeight(msg.Height - 7)
		}

	case TickMsg:
		m = m.updateContent()

		return m, doTick()

	case tea.KeyMsg:
		if !m.getActiveList().SettingFilter() {
			switch {
			case key.Matches(msg, NavKeymap.Quit):
				return m, tea.Quit
			case key.Matches(msg, NavKeymap.Next):
				m.nextTab()
				return m, nil
			case key.Matches(msg, NavKeymap.Prev):
				m.prevTab()
				return m, nil
			}

			if m.activeTab == int(images) {
				switch {
				case key.Matches(msg, ImageKeymap.Delete):
					curItem := m.getSelectedItem()
					if curItem != nil {
						imageId := curItem.(dockerRes).getId()
						err := m.dockerClient.DeleteImage(imageId)
						if err != nil {
							panic(err)
						}
					}
				}

			} else if m.activeTab == int(containers) {
				switch {
				case key.Matches(msg, ContainerKeymap.ToggleListAll):
					m.dockerClient.ToggleContainerListAll()
				case key.Matches(msg, ContainerKeymap.ToggleStartStop):
					log.Println("s pressed")
					curItem := m.getSelectedItem()
					if curItem != nil {
						containerId := curItem.(dockerRes).getId()
						err := m.dockerClient.ToggleStartStopContainer(containerId)
						if err != nil {
							panic(err)
						}
					}
				case key.Matches(msg, ContainerKeymap.Delete):
					curItem := m.getSelectedItem()
					if curItem != nil {
						containerId := curItem.(dockerRes).getId()
						err := m.dockerClient.DeleteContainer(containerId)
						if err != nil {
							panic(err)
						}
					}
				case key.Matches(msg, ContainerKeymap.ToggleLogs):
					m.showLogs = !m.showLogs
				}

			} else {

			}

		}
	}

	var cmds []tea.Cmd
	var Tabcmd tea.Cmd
	m.TabContent[m.activeTab].list, Tabcmd = m.TabContent[m.activeTab].list.Update(msg)
	pagerTemp, pagerCmd := m.logsPager.Update(msg)
	m.logsPager = pagerTemp.(PagerModel)

	cmds = append(cmds, Tabcmd, pagerCmd)

	return m, tea.Batch(cmds...)
}

func tabBorderWithBottom(left, middle, right string) lipgloss.Border {
	border := lipgloss.RoundedBorder()
	border.BottomLeft = left
	border.Bottom = middle
	border.BottomRight = right
	return border
}

func (m Model) View() string {
	doc := strings.Builder{}

	var renderedTabs []string

	for i, t := range m.Tabs {
		var style lipgloss.Style
		isFirst, isLast, isActive := i == 0, i == len(m.Tabs)-1, i == m.activeTab
		if isActive {
			style = activeTabStyle.Copy()
		} else {
			style = inactiveTabStyle.Copy()
		}
		border, _, _, _, _ := style.GetBorder()
		if isFirst && isActive {
			border.BottomLeft = "│"
		} else if isFirst && !isActive {
			border.BottomLeft = "├"
		} else if isLast && isActive {
			border.BottomRight = "└"
		} else if isLast && !isActive {
			border.BottomRight = "┴"
		}

		style = style.Border(border)
		renderedTabs = append(renderedTabs, style.Render(t))
	}

	var row string
	row = lipgloss.JoinHorizontal(lipgloss.Top, renderedTabs...)
	fillerStringLen := windowStyle.GetWidth() - lipgloss.Width(row)
	if fillerStringLen > 0 {
		fillerString := strings.Repeat("─", fillerStringLen+1)
		fillerString += "┐"
		row = lipgloss.JoinHorizontal(lipgloss.Bottom, row, fillerStyle.Render(fillerString))
	}

	list := m.TabContent[m.activeTab].View()
	infobox := m.getInfoBox()

	//TODO: align info box to right edge of the window
	body_with_info := lipgloss.JoinHorizontal(lipgloss.Top, list, infobox)
	// body_with_info = windowStyle.Render(body_with_info)

	doc.WriteString(row)
	doc.WriteString("\n")

	doc.WriteString(body_with_info)
	return docStyle.Render(doc.String())
}

// gets the infobox depending on whether m.showLogs is true or not
func (m Model) getInfoBox() string {
	curItem := m.getSelectedItem()
	if m.showLogs {
		return moreInfoStyle.Render(m.logsPager.View())
	}

	infobox := PopulateInfoBox(tabId(m.activeTab), curItem)
	return moreInfoStyle.Render(infobox)
}

// helpers

func (m Model) updateContent() Model {
	m.TabContent[m.activeTab] = m.TabContent[m.activeTab].updateTab(m.dockerClient, tabId(m.activeTab))
	return m
}

//Util

func (m *Model) nextTab() {
	if m.activeTab == int(volumes) {
		m.activeTab = int(images)
	} else {
		m.activeTab += 1
	}
}

func (m *Model) prevTab() {
	if m.activeTab == int(images) {
		m.activeTab = int(volumes)
	} else {
		m.activeTab -= 1
	}
}

func (m Model) getActiveTab() listModel {
	return m.TabContent[m.activeTab]
}

func (m Model) getActiveList() *list.Model {
	return &m.TabContent[m.activeTab].list
}

func (m Model) getList(index int) *list.Model {
	if index >= len(m.TabContent) {
		panic(fmt.Sprintf("Index %d out of bounds", index))
	}
	return &m.TabContent[index].list
}

func (m Model) getSelectedItem() list.Item {
	return m.TabContent[m.activeTab].list.SelectedItem()
}
