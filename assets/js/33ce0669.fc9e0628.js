"use strict";
(globalThis["webpackChunkwebsite"] = globalThis["webpackChunkwebsite"] || []).push([[68],{

/***/ 1105
(__unused_webpack_module, __webpack_exports__, __webpack_require__) {

// ESM COMPAT FLAG
__webpack_require__.r(__webpack_exports__);

// EXPORTS
__webpack_require__.d(__webpack_exports__, {
  assets: () => (/* binding */ assets),
  contentTitle: () => (/* binding */ contentTitle),
  "default": () => (/* binding */ MDXContent),
  frontMatter: () => (/* binding */ frontMatter),
  metadata: () => (/* reexport */ site_docs_request_examples_md_33c_namespaceObject),
  toc: () => (/* binding */ toc)
});

;// ./.docusaurus/docusaurus-plugin-content-docs/default/site-docs-request-examples-md-33c.json
const site_docs_request_examples_md_33c_namespaceObject = /*#__PURE__*/JSON.parse('{"id":"request-examples","title":"Request Examples","description":"These examples use AWS CLI against s3proxy.","source":"@site/docs/request-examples.md","sourceDirName":".","slug":"/request-examples","permalink":"/docs/request-examples","draft":false,"unlisted":false,"tags":[],"version":"current","sidebarPosition":6,"frontMatter":{"sidebar_position":6},"sidebar":"docsSidebar","previous":{"title":"Routing and Rewrites","permalink":"/docs/providers-and-routing"},"next":{"title":"API Reference","permalink":"/docs/api-reference"}}');
// EXTERNAL MODULE: ./node_modules/.pnpm/react@19.2.6/node_modules/react/jsx-runtime.js
var jsx_runtime = __webpack_require__(1325);
// EXTERNAL MODULE: ./node_modules/.pnpm/@mdx-js+react@3.1.1_@types+react@19.2.14_react@19.2.6/node_modules/@mdx-js/react/lib/index.js
var lib = __webpack_require__(1982);
;// ./docs/request-examples.md


const frontMatter = {
	sidebar_position: 6
};
const contentTitle = 'Request Examples';

const assets = {

};



const toc = [{
  "value": "Common Setup",
  "id": "common-setup",
  "level": 2
}, {
  "value": "List Virtual Buckets",
  "id": "list-virtual-buckets",
  "level": 2
}, {
  "value": "Upload An Object",
  "id": "upload-an-object",
  "level": 2
}, {
  "value": "Read The Object Back",
  "id": "read-the-object-back",
  "level": 2
}, {
  "value": "Head An Object",
  "id": "head-an-object",
  "level": 2
}, {
  "value": "Delete An Object",
  "id": "delete-an-object",
  "level": 2
}, {
  "value": "List Objects In A Bucket View",
  "id": "list-objects-in-a-bucket-view",
  "level": 2
}, {
  "value": "Head A Bucket",
  "id": "head-a-bucket",
  "level": 2
}, {
  "value": "Virtual-Hosted Request Example",
  "id": "virtual-hosted-request-example",
  "level": 2
}, {
  "value": "Unsupported Operations",
  "id": "unsupported-operations",
  "level": 2
}, {
  "value": "Behavior Notes",
  "id": "behavior-notes",
  "level": 2
}];
function _createMdxContent(props) {
  const _components = {
    code: "code",
    h1: "h1",
    h2: "h2",
    header: "header",
    li: "li",
    p: "p",
    pre: "pre",
    ul: "ul",
    ...(0,lib/* useMDXComponents */.R)(),
    ...props.components
  };
  return (0,jsx_runtime.jsxs)(jsx_runtime.Fragment, {
    children: [(0,jsx_runtime.jsx)(_components.header, {
      children: (0,jsx_runtime.jsx)(_components.h1, {
        id: "request-examples",
        children: "Request Examples"
      })
    }), "\n", (0,jsx_runtime.jsxs)(_components.p, {
      children: ["These examples use AWS CLI against ", (0,jsx_runtime.jsx)(_components.code, {
        children: "s3proxy"
      }), "."]
    }), "\n", (0,jsx_runtime.jsxs)(_components.p, {
      children: ["For ", (0,jsx_runtime.jsx)(_components.code, {
        children: "sigv4_static"
      }), ", the proxy validates the client's SigV4 signature. AWS CLI is the simplest way to generate those signed requests without hand-rolling the ", (0,jsx_runtime.jsx)(_components.code, {
        children: "Authorization"
      }), " header."]
    }), "\n", (0,jsx_runtime.jsx)(_components.h2, {
      id: "common-setup",
      children: "Common Setup"
    }), "\n", (0,jsx_runtime.jsx)(_components.p, {
      children: "Set your proxy endpoint and client credentials:"
    }), "\n", (0,jsx_runtime.jsx)(_components.pre, {
      children: (0,jsx_runtime.jsx)(_components.code, {
        className: "language-sh",
        children: "export S3_ENDPOINT=http://127.0.0.1:8080\nexport AWS_ACCESS_KEY_ID=$S3PROXY_CLIENT_ACCESS_KEY\nexport AWS_SECRET_ACCESS_KEY=$S3PROXY_CLIENT_SECRET_KEY\nexport AWS_DEFAULT_REGION=us-east-1\n"
      })
    }), "\n", (0,jsx_runtime.jsxs)(_components.p, {
      children: ["If your proxy is running with ", (0,jsx_runtime.jsx)(_components.code, {
        children: "mode = \"none\""
      }), ", AWS CLI still expects credentials locally, so any placeholder values are fine."]
    }), "\n", (0,jsx_runtime.jsx)(_components.h2, {
      id: "list-virtual-buckets",
      children: "List Virtual Buckets"
    }), "\n", (0,jsx_runtime.jsxs)(_components.p, {
      children: [(0,jsx_runtime.jsx)(_components.code, {
        children: "ListBuckets"
      }), " returns the buckets exposed by proxy config and filtered by ", (0,jsx_runtime.jsx)(_components.code, {
        children: "visible_buckets"
      }), "."]
    }), "\n", (0,jsx_runtime.jsx)(_components.pre, {
      children: (0,jsx_runtime.jsx)(_components.code, {
        className: "language-sh",
        children: "aws --endpoint-url \"$S3_ENDPOINT\" s3api list-buckets\n"
      })
    }), "\n", (0,jsx_runtime.jsx)(_components.h2, {
      id: "upload-an-object",
      children: "Upload An Object"
    }), "\n", (0,jsx_runtime.jsxs)(_components.p, {
      children: ["This request targets the proxy-visible bucket ", (0,jsx_runtime.jsx)(_components.code, {
        children: "images"
      }), ":"]
    }), "\n", (0,jsx_runtime.jsx)(_components.pre, {
      children: (0,jsx_runtime.jsx)(_components.code, {
        className: "language-sh",
        children: "aws --endpoint-url \"$S3_ENDPOINT\" s3api put-object \\\n  --bucket images \\\n  --key cat.jpg \\\n  --body ./cat.jpg\n"
      })
    }), "\n", (0,jsx_runtime.jsxs)(_components.p, {
      children: ["If the matching route rewrites ", (0,jsx_runtime.jsx)(_components.code, {
        children: "bucket = \"images-store\""
      }), ", the object is stored in the backend bucket ", (0,jsx_runtime.jsx)(_components.code, {
        children: "images-store"
      }), " even though the client addressed ", (0,jsx_runtime.jsx)(_components.code, {
        children: "images"
      }), "."]
    }), "\n", (0,jsx_runtime.jsx)(_components.h2, {
      id: "read-the-object-back",
      children: "Read The Object Back"
    }), "\n", (0,jsx_runtime.jsx)(_components.pre, {
      children: (0,jsx_runtime.jsx)(_components.code, {
        className: "language-sh",
        children: "aws --endpoint-url \"$S3_ENDPOINT\" s3api get-object \\\n  --bucket images \\\n  --key cat.jpg \\\n  ./cat.out.jpg\n"
      })
    }), "\n", (0,jsx_runtime.jsxs)(_components.p, {
      children: ["For a route with ", (0,jsx_runtime.jsx)(_components.code, {
        children: "read_preference = \"ordered_failover\""
      }), ", the proxy will try later destinations only on transport errors, timeouts, or upstream ", (0,jsx_runtime.jsx)(_components.code, {
        children: "5xx"
      }), "."]
    }), "\n", (0,jsx_runtime.jsx)(_components.h2, {
      id: "head-an-object",
      children: "Head An Object"
    }), "\n", (0,jsx_runtime.jsx)(_components.pre, {
      children: (0,jsx_runtime.jsx)(_components.code, {
        className: "language-sh",
        children: "aws --endpoint-url \"$S3_ENDPOINT\" s3api head-object \\\n  --bucket images \\\n  --key cat.jpg\n"
      })
    }), "\n", (0,jsx_runtime.jsx)(_components.h2, {
      id: "delete-an-object",
      children: "Delete An Object"
    }), "\n", (0,jsx_runtime.jsx)(_components.pre, {
      children: (0,jsx_runtime.jsx)(_components.code, {
        className: "language-sh",
        children: "aws --endpoint-url \"$S3_ENDPOINT\" s3api delete-object \\\n  --bucket images \\\n  --key cat.jpg\n"
      })
    }), "\n", (0,jsx_runtime.jsxs)(_components.p, {
      children: ["On a route with ", (0,jsx_runtime.jsx)(_components.code, {
        children: "dispatch = \"all\""
      }), ", ", (0,jsx_runtime.jsx)(_components.code, {
        children: "DeleteObject"
      }), " is replayed to every configured destination."]
    }), "\n", (0,jsx_runtime.jsx)(_components.h2, {
      id: "list-objects-in-a-bucket-view",
      children: "List Objects In A Bucket View"
    }), "\n", (0,jsx_runtime.jsx)(_components.pre, {
      children: (0,jsx_runtime.jsx)(_components.code, {
        className: "language-sh",
        children: "aws --endpoint-url \"$S3_ENDPOINT\" s3api list-objects-v2 \\\n  --bucket images\n"
      })
    }), "\n", (0,jsx_runtime.jsxs)(_components.p, {
      children: ["In v1, ", (0,jsx_runtime.jsx)(_components.code, {
        children: "ListObjectsV2"
      }), " always comes from one effective backend. The proxy does not merge pagination across multiple destinations."]
    }), "\n", (0,jsx_runtime.jsx)(_components.h2, {
      id: "head-a-bucket",
      children: "Head A Bucket"
    }), "\n", (0,jsx_runtime.jsx)(_components.pre, {
      children: (0,jsx_runtime.jsx)(_components.code, {
        className: "language-sh",
        children: "aws --endpoint-url \"$S3_ENDPOINT\" s3api head-bucket \\\n  --bucket images\n"
      })
    }), "\n", (0,jsx_runtime.jsx)(_components.h2, {
      id: "virtual-hosted-request-example",
      children: "Virtual-Hosted Request Example"
    }), "\n", (0,jsx_runtime.jsxs)(_components.p, {
      children: ["If the listener enables ", (0,jsx_runtime.jsx)(_components.code, {
        children: "virtual_hosted = true"
      }), " with ", (0,jsx_runtime.jsx)(_components.code, {
        children: "host_suffixes = [\"s3proxy.example.com\"]"
      }), ", a client can address a bucket through the host instead of the path."]
    }), "\n", (0,jsx_runtime.jsx)(_components.p, {
      children: "Example request shape:"
    }), "\n", (0,jsx_runtime.jsx)(_components.pre, {
      children: (0,jsx_runtime.jsx)(_components.code, {
        className: "language-http",
        children: "GET /cat.jpg HTTP/1.1\nHost: images.s3proxy.example.com\nAuthorization: AWS4-HMAC-SHA256 ...\n"
      })
    }), "\n", (0,jsx_runtime.jsx)(_components.h2, {
      id: "unsupported-operations",
      children: "Unsupported Operations"
    }), "\n", (0,jsx_runtime.jsxs)(_components.p, {
      children: ["Multipart upload operations are not implemented in v1. ", (0,jsx_runtime.jsx)(_components.code, {
        children: "CopyObject"
      }), " is also rejected."]
    }), "\n", (0,jsx_runtime.jsxs)(_components.p, {
      children: ["The proxy returns an S3-compatible ", (0,jsx_runtime.jsx)(_components.code, {
        children: "NotImplemented"
      }), " error for those requests instead of attempting partial support."]
    }), "\n", (0,jsx_runtime.jsx)(_components.h2, {
      id: "behavior-notes",
      children: "Behavior Notes"
    }), "\n", (0,jsx_runtime.jsxs)(_components.ul, {
      children: ["\n", (0,jsx_runtime.jsx)(_components.li, {
        children: "direct single-destination reads do not fail over anywhere else"
      }), "\n", (0,jsx_runtime.jsxs)(_components.li, {
        children: [(0,jsx_runtime.jsx)(_components.code, {
          children: "dispatch = \"all\""
        }), " writes must succeed on every destination"]
      }), "\n", (0,jsx_runtime.jsxs)(_components.li, {
        children: [(0,jsx_runtime.jsx)(_components.code, {
          children: "ordered_failover"
        }), " does not fail over on ", (0,jsx_runtime.jsx)(_components.code, {
          children: "404"
        }), ", ", (0,jsx_runtime.jsx)(_components.code, {
          children: "NoSuchKey"
        }), ", or ", (0,jsx_runtime.jsx)(_components.code, {
          children: "NoSuchBucket"
        })]
      }), "\n", (0,jsx_runtime.jsxs)(_components.li, {
        children: ["request paths preserve escaped bytes, so ", (0,jsx_runtime.jsx)(_components.code, {
          children: "%2F"
        }), " remains distinct from ", (0,jsx_runtime.jsx)(_components.code, {
          children: "/"
        })]
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