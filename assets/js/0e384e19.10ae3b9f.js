"use strict";
(globalThis["webpackChunkwebsite"] = globalThis["webpackChunkwebsite"] || []).push([[976],{

/***/ 3394
(__unused_webpack_module, __webpack_exports__, __webpack_require__) {

// ESM COMPAT FLAG
__webpack_require__.r(__webpack_exports__);

// EXPORTS
__webpack_require__.d(__webpack_exports__, {
  assets: () => (/* binding */ assets),
  contentTitle: () => (/* binding */ contentTitle),
  "default": () => (/* binding */ MDXContent),
  frontMatter: () => (/* binding */ frontMatter),
  metadata: () => (/* reexport */ site_docs_intro_md_0e3_namespaceObject),
  toc: () => (/* binding */ toc)
});

;// ./.docusaurus/docusaurus-plugin-content-docs/default/site-docs-intro-md-0e3.json
const site_docs_intro_md_0e3_namespaceObject = /*#__PURE__*/JSON.parse('{"id":"intro","title":"s3proxy","description":"s3proxy is a Go service that accepts S3-compatible requests and forwards them to one or more configured S3-compatible backends.","source":"@site/docs/intro.md","sourceDirName":".","slug":"/intro","permalink":"/docs/intro","draft":false,"unlisted":false,"tags":[],"version":"current","sidebarPosition":1,"frontMatter":{"sidebar_position":1},"sidebar":"docsSidebar","next":{"title":"Quickstart","permalink":"/docs/quickstart"}}');
// EXTERNAL MODULE: ./node_modules/.pnpm/react@19.2.6/node_modules/react/jsx-runtime.js
var jsx_runtime = __webpack_require__(1325);
// EXTERNAL MODULE: ./node_modules/.pnpm/@mdx-js+react@3.1.1_@types+react@19.2.14_react@19.2.6/node_modules/@mdx-js/react/lib/index.js
var lib = __webpack_require__(1982);
;// ./docs/intro.md


const frontMatter = {
	sidebar_position: 1
};
const contentTitle = 's3proxy';

const assets = {

};



const toc = [{
  "value": "What It Solves",
  "id": "what-it-solves",
  "level": 2
}, {
  "value": "Current V1 Scope",
  "id": "current-v1-scope",
  "level": 2
}, {
  "value": "Non-Goals In V1",
  "id": "non-goals-in-v1",
  "level": 2
}, {
  "value": "How A Request Flows",
  "id": "how-a-request-flows",
  "level": 2
}, {
  "value": "Key Behaviors",
  "id": "key-behaviors",
  "level": 2
}, {
  "value": "Documentation Map",
  "id": "documentation-map",
  "level": 2
}, {
  "value": "Design Document",
  "id": "design-document",
  "level": 2
}];
function _createMdxContent(props) {
  const _components = {
    a: "a",
    code: "code",
    h1: "h1",
    h2: "h2",
    header: "header",
    li: "li",
    ol: "ol",
    p: "p",
    ul: "ul",
    ...(0,lib/* useMDXComponents */.R)(),
    ...props.components
  };
  return (0,jsx_runtime.jsxs)(jsx_runtime.Fragment, {
    children: [(0,jsx_runtime.jsx)(_components.header, {
      children: (0,jsx_runtime.jsx)(_components.h1, {
        id: "s3proxy",
        children: "s3proxy"
      })
    }), "\n", (0,jsx_runtime.jsxs)(_components.p, {
      children: [(0,jsx_runtime.jsx)(_components.code, {
        children: "s3proxy"
      }), " is a Go service that accepts S3-compatible requests and forwards them to one or more configured S3-compatible backends."]
    }), "\n", (0,jsx_runtime.jsx)(_components.p, {
      children: "It sits at the S3 protocol boundary: the proxy can authenticate the client, classify the S3 operation, match a route from the request path, bucket name, host, and operation, rewrite the bucket or key, then sign outbound requests with backend credentials."
    }), "\n", (0,jsx_runtime.jsx)(_components.h2, {
      id: "what-it-solves",
      children: "What It Solves"
    }), "\n", (0,jsx_runtime.jsxs)(_components.p, {
      children: [(0,jsx_runtime.jsx)(_components.code, {
        children: "s3proxy"
      }), " is useful when you want one stable S3 endpoint while the actual storage layout behind it is more opinionated than a single bucket on a single backend."]
    }), "\n", (0,jsx_runtime.jsx)(_components.p, {
      children: "Typical reasons to use it:"
    }), "\n", (0,jsx_runtime.jsxs)(_components.ul, {
      children: ["\n", (0,jsx_runtime.jsx)(_components.li, {
        children: "expose path-based or host-based virtual views over existing buckets"
      }), "\n", (0,jsx_runtime.jsx)(_components.li, {
        children: "replicate writes to more than one S3-compatible backend"
      }), "\n", (0,jsx_runtime.jsxs)(_components.li, {
        children: ["fail reads over to a secondary backend on transport errors and upstream ", (0,jsx_runtime.jsx)(_components.code, {
          children: "5xx"
        })]
      }), "\n", (0,jsx_runtime.jsx)(_components.li, {
        children: "keep client credentials separate from backend credentials"
      }), "\n", (0,jsx_runtime.jsxs)(_components.li, {
        children: ["present a proxy-defined ", (0,jsx_runtime.jsx)(_components.code, {
          children: "ListBuckets"
        }), " view instead of exposing upstream bucket discovery"]
      }), "\n"]
    }), "\n", (0,jsx_runtime.jsx)(_components.h2, {
      id: "current-v1-scope",
      children: "Current V1 Scope"
    }), "\n", (0,jsx_runtime.jsx)(_components.p, {
      children: "Supported operations:"
    }), "\n", (0,jsx_runtime.jsxs)(_components.ul, {
      children: ["\n", (0,jsx_runtime.jsx)(_components.li, {
        children: (0,jsx_runtime.jsx)(_components.code, {
          children: "GetObject"
        })
      }), "\n", (0,jsx_runtime.jsx)(_components.li, {
        children: (0,jsx_runtime.jsx)(_components.code, {
          children: "HeadObject"
        })
      }), "\n", (0,jsx_runtime.jsx)(_components.li, {
        children: (0,jsx_runtime.jsx)(_components.code, {
          children: "PutObject"
        })
      }), "\n", (0,jsx_runtime.jsx)(_components.li, {
        children: (0,jsx_runtime.jsx)(_components.code, {
          children: "DeleteObject"
        })
      }), "\n", (0,jsx_runtime.jsx)(_components.li, {
        children: (0,jsx_runtime.jsx)(_components.code, {
          children: "HeadBucket"
        })
      }), "\n", (0,jsx_runtime.jsx)(_components.li, {
        children: (0,jsx_runtime.jsx)(_components.code, {
          children: "ListObjectsV2"
        })
      }), "\n", (0,jsx_runtime.jsx)(_components.li, {
        children: (0,jsx_runtime.jsx)(_components.code, {
          children: "ListBuckets"
        })
      }), "\n"]
    }), "\n", (0,jsx_runtime.jsx)(_components.p, {
      children: "Supported inbound auth modes:"
    }), "\n", (0,jsx_runtime.jsxs)(_components.ul, {
      children: ["\n", (0,jsx_runtime.jsx)(_components.li, {
        children: (0,jsx_runtime.jsx)(_components.code, {
          children: "none"
        })
      }), "\n", (0,jsx_runtime.jsx)(_components.li, {
        children: (0,jsx_runtime.jsx)(_components.code, {
          children: "sigv4_static"
        })
      }), "\n"]
    }), "\n", (0,jsx_runtime.jsx)(_components.p, {
      children: "Supported routing and rewrite features:"
    }), "\n", (0,jsx_runtime.jsxs)(_components.ul, {
      children: ["\n", (0,jsx_runtime.jsx)(_components.li, {
        children: "path-style and virtual-hosted addressing"
      }), "\n", (0,jsx_runtime.jsxs)(_components.li, {
        children: ["parsers: ", (0,jsx_runtime.jsx)(_components.code, {
          children: "path_prefix"
        }), ", ", (0,jsx_runtime.jsx)(_components.code, {
          children: "bucket_exact"
        }), ", ", (0,jsx_runtime.jsx)(_components.code, {
          children: "bucket_regex"
        }), ", ", (0,jsx_runtime.jsx)(_components.code, {
          children: "host_suffix"
        })]
      }), "\n", (0,jsx_runtime.jsxs)(_components.li, {
        children: ["ordered route evaluation with ", (0,jsx_runtime.jsx)(_components.code, {
          children: "stop"
        }), " and ", (0,jsx_runtime.jsx)(_components.code, {
          children: "continue"
        })]
      }), "\n", (0,jsx_runtime.jsxs)(_components.li, {
        children: ["destination dispatch modes: ", (0,jsx_runtime.jsx)(_components.code, {
          children: "first"
        }), " and ", (0,jsx_runtime.jsx)(_components.code, {
          children: "all"
        })]
      }), "\n", (0,jsx_runtime.jsxs)(_components.li, {
        children: ["read preferences: ", (0,jsx_runtime.jsx)(_components.code, {
          children: "first"
        }), ", ", (0,jsx_runtime.jsx)(_components.code, {
          children: "random"
        }), ", ", (0,jsx_runtime.jsx)(_components.code, {
          children: "hash"
        }), ", ", (0,jsx_runtime.jsx)(_components.code, {
          children: "ordered_failover"
        })]
      }), "\n", (0,jsx_runtime.jsxs)(_components.li, {
        children: ["rewrite rules: ", (0,jsx_runtime.jsx)(_components.code, {
          children: "strip_path_prefix"
        }), ", ", (0,jsx_runtime.jsx)(_components.code, {
          children: "strip_key_prefix"
        }), ", ", (0,jsx_runtime.jsx)(_components.code, {
          children: "prepend_key_prefix"
        }), ", ", (0,jsx_runtime.jsx)(_components.code, {
          children: "bucket"
        }), ", ", (0,jsx_runtime.jsx)(_components.code, {
          children: "key_template"
        })]
      }), "\n"]
    }), "\n", (0,jsx_runtime.jsx)(_components.h2, {
      id: "non-goals-in-v1",
      children: "Non-Goals In V1"
    }), "\n", (0,jsx_runtime.jsx)(_components.p, {
      children: "The project intentionally does not implement these yet:"
    }), "\n", (0,jsx_runtime.jsxs)(_components.ul, {
      children: ["\n", (0,jsx_runtime.jsx)(_components.li, {
        children: "multipart upload support"
      }), "\n", (0,jsx_runtime.jsxs)(_components.li, {
        children: ["merged multi-backend ", (0,jsx_runtime.jsx)(_components.code, {
          children: "ListObjects"
        }), " pagination"]
      }), "\n", (0,jsx_runtime.jsx)(_components.li, {
        children: "credential generation or rotation"
      }), "\n", (0,jsx_runtime.jsx)(_components.li, {
        children: "non-S3 backend types"
      }), "\n", (0,jsx_runtime.jsx)(_components.li, {
        children: "hot config reload"
      }), "\n", (0,jsx_runtime.jsx)(_components.li, {
        children: "full presigned URL feature parity"
      }), "\n"]
    }), "\n", (0,jsx_runtime.jsxs)(_components.p, {
      children: ["Multipart-related operations and ", (0,jsx_runtime.jsx)(_components.code, {
        children: "CopyObject"
      }), " return an S3-compatible ", (0,jsx_runtime.jsx)(_components.code, {
        children: "NotImplemented"
      }), " error."]
    }), "\n", (0,jsx_runtime.jsx)(_components.h2, {
      id: "how-a-request-flows",
      children: "How A Request Flows"
    }), "\n", (0,jsx_runtime.jsxs)(_components.ol, {
      children: ["\n", (0,jsx_runtime.jsx)(_components.li, {
        children: "The proxy receives an S3-compatible HTTP request."
      }), "\n", (0,jsx_runtime.jsx)(_components.li, {
        children: "It derives a normalized request context from host, path, query, and method."
      }), "\n", (0,jsx_runtime.jsx)(_components.li, {
        children: "If auth is enabled, it validates the inbound SigV4 signature against configured client credentials."
      }), "\n", (0,jsx_runtime.jsx)(_components.li, {
        children: "It classifies the operation and resolves one or more matching routes."
      }), "\n", (0,jsx_runtime.jsx)(_components.li, {
        children: "It applies any configured bucket or key rewrites."
      }), "\n", (0,jsx_runtime.jsx)(_components.li, {
        children: "It builds outbound S3 requests to the selected target backends."
      }), "\n", (0,jsx_runtime.jsx)(_components.li, {
        children: "It signs those outbound requests with the target credentials and returns an S3-compatible response."
      }), "\n"]
    }), "\n", (0,jsx_runtime.jsx)(_components.h2, {
      id: "key-behaviors",
      children: "Key Behaviors"
    }), "\n", (0,jsx_runtime.jsxs)(_components.ul, {
      children: ["\n", (0,jsx_runtime.jsxs)(_components.li, {
        children: [(0,jsx_runtime.jsx)(_components.code, {
          children: "ListBuckets"
        }), " is virtual. It returns proxy-defined buckets visible to the authenticated client, not upstream bucket discovery."]
      }), "\n", (0,jsx_runtime.jsx)(_components.li, {
        children: "Reads never fan out in v1. Even when a route has multiple destinations, one effective backend is selected per read."
      }), "\n", (0,jsx_runtime.jsxs)(_components.li, {
        children: ["For ", (0,jsx_runtime.jsx)(_components.code, {
          children: "dispatch = \"all\""
        }), ", write request bodies are buffered in memory before they are replayed to each destination."]
      }), "\n", (0,jsx_runtime.jsxs)(_components.li, {
        children: [(0,jsx_runtime.jsx)(_components.code, {
          children: "ordered_failover"
        }), " only fails over on transport errors, timeouts, and upstream ", (0,jsx_runtime.jsx)(_components.code, {
          children: "5xx"
        }), ". It does not fail over on ", (0,jsx_runtime.jsx)(_components.code, {
          children: "404"
        }), ", ", (0,jsx_runtime.jsx)(_components.code, {
          children: "NoSuchKey"
        }), ", or ", (0,jsx_runtime.jsx)(_components.code, {
          children: "NoSuchBucket"
        }), "."]
      }), "\n", (0,jsx_runtime.jsxs)(_components.li, {
        children: [(0,jsx_runtime.jsx)(_components.code, {
          children: "path_prefix"
        }), " matching is strict: ", (0,jsx_runtime.jsx)(_components.code, {
          children: "RawPath == prefix"
        }), " or ", (0,jsx_runtime.jsx)(_components.code, {
          children: "RawPath"
        }), " starts with ", (0,jsx_runtime.jsx)(_components.code, {
          children: "prefix + \"/\""
        }), "."]
      }), "\n"]
    }), "\n", (0,jsx_runtime.jsx)(_components.h2, {
      id: "documentation-map",
      children: "Documentation Map"
    }), "\n", (0,jsx_runtime.jsxs)(_components.ul, {
      children: ["\n", (0,jsx_runtime.jsxs)(_components.li, {
        children: [(0,jsx_runtime.jsx)(_components.a, {
          href: "/docs/quickstart",
          children: "Quickstart"
        }), " for a first local setup"]
      }), "\n", (0,jsx_runtime.jsxs)(_components.li, {
        children: [(0,jsx_runtime.jsx)(_components.a, {
          href: "/docs/configuration",
          children: "Configuration"
        }), " for the HCL model and validation rules"]
      }), "\n", (0,jsx_runtime.jsxs)(_components.li, {
        children: [(0,jsx_runtime.jsx)(_components.a, {
          href: "/docs/config-examples",
          children: "Config Examples"
        }), " for complete sample configs"]
      }), "\n", (0,jsx_runtime.jsxs)(_components.li, {
        children: [(0,jsx_runtime.jsx)(_components.a, {
          href: "/docs/providers-and-routing",
          children: "Providers and Routing"
        }), " for route evaluation, parsers, rewrites, and failover"]
      }), "\n", (0,jsx_runtime.jsxs)(_components.li, {
        children: [(0,jsx_runtime.jsx)(_components.a, {
          href: "/docs/request-examples",
          children: "Request Examples"
        }), " for AWS CLI examples against the proxy"]
      }), "\n", (0,jsx_runtime.jsxs)(_components.li, {
        children: [(0,jsx_runtime.jsx)(_components.a, {
          href: "/docs/api-reference",
          children: "API Reference"
        }), " for operation coverage and request behavior"]
      }), "\n", (0,jsx_runtime.jsxs)(_components.li, {
        children: [(0,jsx_runtime.jsx)(_components.a, {
          href: "/docs/operations",
          children: "Operations"
        }), " for build, test, sandbox, and runtime commands"]
      }), "\n", (0,jsx_runtime.jsxs)(_components.li, {
        children: [(0,jsx_runtime.jsx)(_components.a, {
          href: "/docs/deployment",
          children: "Deployment"
        }), " for Docker, service management, and rollout guidance"]
      }), "\n"]
    }), "\n", (0,jsx_runtime.jsx)(_components.h2, {
      id: "design-document",
      children: "Design Document"
    }), "\n", (0,jsx_runtime.jsx)(_components.p, {
      children: "For the full implementation rationale and deferred feature list, see the repository design doc:"
    }), "\n", (0,jsx_runtime.jsxs)(_components.ul, {
      children: ["\n", (0,jsx_runtime.jsx)(_components.li, {
        children: (0,jsx_runtime.jsx)(_components.a, {
          href: "https://github.com/egose/s3proxy/blob/main/docs/design.md",
          children: "github.com/egose/s3proxy/blob/main/docs/design.md"
        })
      }), "\n"]
    })]
  });
}
function MDXContent(props = {}) {
  const {wrapper: MDXLayout} = {
    ...(0,lib/* useMDXComponents */.R)(),
    ...props.components
  };
  return MDXLayout ? (0,jsx_runtime.jsx)(MDXLayout, {
    ...props,
    children: (0,jsx_runtime.jsx)(_createMdxContent, {
      ...props
    })
  }) : _createMdxContent(props);
}



/***/ },

/***/ 1982
(__unused_webpack___webpack_module__, __webpack_exports__, __webpack_require__) {

/* harmony export */ __webpack_require__.d(__webpack_exports__, {
/* harmony export */   R: () => (/* binding */ useMDXComponents),
/* harmony export */   x: () => (/* binding */ MDXProvider)
/* harmony export */ });
/* harmony import */ var react__WEBPACK_IMPORTED_MODULE_0__ = __webpack_require__(489);
/**
 * @import {MDXComponents} from 'mdx/types.js'
 * @import {Component, ReactElement, ReactNode} from 'react'
 */

/**
 * @callback MergeComponents
 *   Custom merge function.
 * @param {Readonly<MDXComponents>} currentComponents
 *   Current components from the context.
 * @returns {MDXComponents}
 *   Additional components.
 *
 * @typedef Props
 *   Configuration for `MDXProvider`.
 * @property {ReactNode | null | undefined} [children]
 *   Children (optional).
 * @property {Readonly<MDXComponents> | MergeComponents | null | undefined} [components]
 *   Additional components to use or a function that creates them (optional).
 * @property {boolean | null | undefined} [disableParentContext=false]
 *   Turn off outer component context (default: `false`).
 */



/** @type {Readonly<MDXComponents>} */
const emptyComponents = {}

const MDXContext = react__WEBPACK_IMPORTED_MODULE_0__.createContext(emptyComponents)

/**
 * Get current components from the MDX Context.
 *
 * @param {Readonly<MDXComponents> | MergeComponents | null | undefined} [components]
 *   Additional components to use or a function that creates them (optional).
 * @returns {MDXComponents}
 *   Current components.
 */
function useMDXComponents(components) {
  const contextComponents = react__WEBPACK_IMPORTED_MODULE_0__.useContext(MDXContext)

  // Memoize to avoid unnecessary top-level context changes
  return react__WEBPACK_IMPORTED_MODULE_0__.useMemo(
    function () {
      // Custom merge via a function prop
      if (typeof components === 'function') {
        return components(contextComponents)
      }

      return {...contextComponents, ...components}
    },
    [contextComponents, components]
  )
}

/**
 * Provider for MDX context.
 *
 * @param {Readonly<Props>} properties
 *   Properties.
 * @returns {ReactElement}
 *   Element.
 * @satisfies {Component}
 */
function MDXProvider(properties) {
  /** @type {Readonly<MDXComponents>} */
  let allComponents

  if (properties.disableParentContext) {
    allComponents =
      typeof properties.components === 'function'
        ? properties.components(emptyComponents)
        : properties.components || emptyComponents
  } else {
    allComponents = useMDXComponents(properties.components)
  }

  return react__WEBPACK_IMPORTED_MODULE_0__.createElement(
    MDXContext.Provider,
    {value: allComponents},
    properties.children
  )
}


/***/ }

}]);