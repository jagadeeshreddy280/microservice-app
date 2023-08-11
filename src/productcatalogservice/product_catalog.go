// Copyright 2023 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"context"
	"errors"
	"strings"
	"time"

	pb "github.com/GoogleCloudPlatform/microservices-demo/src/productcatalogservice/genproto"
	"google.golang.org/grpc/codes"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/status"
)

type productCatalog struct {
	catalog pb.ListProductsResponse
	pb.UnimplementedProductCatalogServiceServer
}

func (p *productCatalog) Check(ctx context.Context, req *healthpb.HealthCheckRequest) (*healthpb.HealthCheckResponse, error) {
	return &healthpb.HealthCheckResponse{Status: healthpb.HealthCheckResponse_SERVING}, nil
}

func (p *productCatalog) Watch(req *healthpb.HealthCheckRequest, ws healthpb.Health_WatchServer) error {
	return status.Errorf(codes.Unimplemented, "health check via Watch not implemented")
}

func (p *productCatalog) ListProducts(ctx context.Context, req *pb.ListProductsRequest) (*pb.ListProductsResponse, error) {
	time.Sleep(extraLatency)
	products := p.parseCatalog()
	translateProductsInPlace(ctx, getGoogleCloudProjectId(), req.Language, products)
	return &pb.ListProductsResponse{Products: products}, nil
}

func (p *productCatalog) GetProduct(ctx context.Context, req *pb.GetProductRequest) (*pb.Product, error) {
	time.Sleep(extraLatency)

	var found *pb.Product
	products := p.parseCatalog()
	translateProductsInPlace(ctx, getGoogleCloudProjectId(), req.Language, products)
	for i := 0; i < len(p.parseCatalog()); i++ {
		if req.Id == p.parseCatalog()[i].Id {
			found = p.parseCatalog()[i]
		}
	}

	if found == nil {
		return nil, status.Errorf(codes.NotFound, "no product with ID %s", req.Id)
	}
	return found, nil
}

func (p *productCatalog) SearchProducts(ctx context.Context, req *pb.SearchProductsRequest) (*pb.SearchProductsResponse, error) {
	time.Sleep(extraLatency)

	var ps []*pb.Product
	for _, product := range p.parseCatalog() {
		if strings.Contains(strings.ToLower(product.Name), strings.ToLower(req.Query)) ||
			strings.Contains(strings.ToLower(product.Description), strings.ToLower(req.Query)) {
			ps = append(ps, product)
		}
	}

	return &pb.SearchProductsResponse{Results: ps}, nil
}

func (p *productCatalog) parseCatalog() []*pb.Product {
	if reloadCatalog || len(p.catalog.Products) == 0 {
		err := readCatalogFile(&p.catalog)
		if err != nil {
			return []*pb.Product{}
		}
	}

	return p.catalog.Products
}

func translateProductsInPlace(ctx context.Context, projectId, targetLangCode string, products []*pb.Product) error {
	// Handle English to English translations.
	if targetLangCode == "en" {
		return nil
	}
	// Ensure the target language is supported.
	supportedTargetLangs := map[string]bool{
		"es": true,
		"hi": true,
		"ja": true,
		"pt": true,
	}
	if !supportedTargetLangs[targetLangCode] {
		return errors.New("Unsupported target language")
	}
	// Collect the strings to be translated.
	stringsToTranslate := make([]string, len(products)*2)
	for i, product := range products {
		stringsToTranslate[i*2] = product.Name
		stringsToTranslate[(i*2)+1] = product.Description
	}
	// Translate the strings.
	translatedStrings, err := translateStrings(ctx, projectId, targetLangCode, stringsToTranslate)
	if err != nil {
		return err
	}
	// Update products in-place with translations.
	for i, product := range products {
		product.Name = translatedStrings[i*2]
		product.Description = translatedStrings[(i*2)+1]
	}
	return nil
}
