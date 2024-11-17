package graph

import (
	"github.com/mbaksheev/clickhouse-table-graph/table"
)

type Graph struct {
	InitialTable table.Key
	Links        []Link
	tables       map[table.Key]table.Info
}

func (g *Graph) TableInfo(key table.Key) (table.Info, bool) {
	info, exists := g.tables[key]
	return info, exists
}

type Builder interface {
	AddTable(table table.Info)
	Build(TableKey table.Key) (*Graph, error)
}

func New() Builder {
	return &builder{
		nodes:  make(map[table.Key]*graphNode),
		tables: make(map[table.Key]table.Info),
	}
}

type builder struct {
	nodes  map[table.Key]*graphNode
	tables map[table.Key]table.Info
}

type stackItem struct {
	tableKey   table.Key
	isToParent bool
}

func (g *builder) Build(initialTableKey table.Key) (*Graph, error) {
	// use depth-first search to find all links for the specified initialTableKey
	graphLinks := make([]Link, 0)
	visited := make(map[table.Key]bool)
	stack := []stackItem{{tableKey: initialTableKey, isToParent: false}}

	for len(stack) > 0 {
		currentStackItem := stack[len(stack)-1]
		currentKey := currentStackItem.tableKey
		stack = stack[:len(stack)-1]

		if visited[currentKey] {
			continue
		}

		visited[currentKey] = true

		node, exists := g.nodes[currentKey]
		if !exists {
			continue
		}

		for _, toLink := range node.toLinks {
			if !visited[toLink] && !currentStackItem.isToParent {
				graphLinks = append(graphLinks, Link{
					FromTableKey: currentKey,
					ToTableKey:   toLink,
				})
				stack = append(stack, stackItem{tableKey: toLink, isToParent: false})
			}

		}

		for _, link := range node.fromLinks {
			if !visited[link] {
				graphLinks = append(graphLinks, Link{
					FromTableKey: link,
					ToTableKey:   currentKey,
				})
				stack = append(stack, stackItem{tableKey: link, isToParent: true})
			}
		}
	}
	return &Graph{
			InitialTable: initialTableKey,
			Links:        graphLinks,
			tables:       g.tables,
		},
		nil
}

func (g *builder) AddTable(tableInfo table.Info) {
	g.tables[tableInfo.Key] = tableInfo
	newNode := createGraphNode(tableInfo)

	if node, exists := g.nodes[tableInfo.Key]; exists {
		node.fromLinks = append(node.fromLinks, newNode.fromLinks...)
		node.toLinks = append(node.toLinks, newNode.toLinks...)
	} else {
		g.nodes[tableInfo.Key] = &newNode
	}
	for _, fromLink := range newNode.fromLinks {
		if node, exists := g.nodes[fromLink]; exists {
			node.toLinks = append(node.toLinks, tableInfo.Key)
		} else {
			g.nodes[fromLink] = &graphNode{
				fromLinks: make([]table.Key, 0),
				toLinks:   []table.Key{tableInfo.Key},
			}
		}
	}
	for _, toLink := range newNode.toLinks {
		if node, exists := g.nodes[toLink]; exists {
			node.fromLinks = append(node.fromLinks, tableInfo.Key)
		} else {
			g.nodes[toLink] = &graphNode{
				fromLinks: []table.Key{tableInfo.Key},
				toLinks:   make([]table.Key, 0),
			}
		}
	}
}
