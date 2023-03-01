package client

import (
	"context"
	"errors"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

type Product struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Plans       []*Plan `json:"plans"`
}

type Plan struct {
	ID         string  `json:"id"`
	ProductID  string  `json:"productId"`
	RegionCode string  `json:"regionCode"`
	Price      float64 `json:"price"`
	MilliCPU   int64   `json:"milliCPU"`
	MemoryGB   int64   `json:"memoryGB"`
	StorageGB  int64   `json:"storageGB"`
}

type ProductsResponse struct {
	Products []*Product `json:"products"`
}

func (c *Client) GetProducts(ctx context.Context) ([]*Product, error) {
	tflog.Trace(ctx, "Client.GetProducts")
	req := map[string]interface{}{
		"operationName": "GetProducts",
		"query":         ProductsQuery,
	}
	var resp Response[ProductsResponse]
	if err := c.do(ctx, req, &resp); err != nil {
		return nil, err
	}
	if len(resp.Errors) > 0 {
		return nil, resp.Errors[0]
	}
	if resp.Data == nil {
		return nil, errors.New("no response found")
	}
	return resp.Data.Products, nil
}
